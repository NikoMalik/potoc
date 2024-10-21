package config

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/NikoMalik/potoc/internal/logger"
	grpcMiddleware "github.com/grpc-ecosystem/go-grpc-middleware"
	"github.com/spf13/viper"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"google.golang.org/grpc"
)

const (
	Local = "local"
	Prod  = "prod"
)

type Config struct {
	Env      string  `mapstructure:"env"`
	Server   *Server `mapstructure:"server"`
	DB       *DB     `mapstructure:"db"`
	LogLevel string  `mapstructure:"log_level"`
}

type Server struct {
	Host                  string              `mapstructure:"host"`
	Port                  string              `mapstructure:"port"`
	Opts                  []grpc.ServerOption `mapstructure:"opts"`
	MaxStreams            uint32              `mapstructure:"max_streams"`
	WriteBufferSize       int                 `mapstructure:"write_buffer_size"`
	ReadBufferSize        int                 `mapstructure:"read_buffer_size"`
	InitialWindowSize     int32               `mapstructure:"initial_window_size"`
	InitialConnWindowSize int32               `mapstructure:"initial_conn_window_size"`
	MaxHeaderListSize     uint32              `mapstructure:"max_header_list_size"`
	MaxRecvMsgSize        int                 `mapstructure:"max_recv_msg_size"`
}

type DB struct {
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	User     string `mapstructure:"user"`
	Password string `mapstructure:"password"`
	Name     string `mapstructure:"db_name"`
	SSLMode  string `mapstructure:"ssl_mode"`
}

func OpenLoad(env string) (*Config, error) {
	viper.SetConfigName(env)
	viper.SetConfigType("yaml")
	viper.AddConfigPath("configs/")

	viper.AutomaticEnv()
	if err := viper.ReadInConfig(); err != nil {
		return nil, err
	}

	for _, k := range viper.AllKeys() {
		value := viper.GetString(k)
		if strings.HasPrefix(value, "${") && strings.HasSuffix(value, "}") {
			viper.Set(k, getEnvOrPanic(strings.TrimSuffix(strings.TrimPrefix(value, "${"), "}")))
		}
	}

	var config *Config
	if err := viper.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("unable to decode into struct, %v", err)
	}

	return config, nil
}

func getEnvOrPanic(env string) string {
	res := os.Getenv(env)

	if len(res) <= 0 {
		panic(fmt.Sprintf("env variable %s not found", env))
	}

	return res
}

func NewConfig(env string) (*Config, error) {
	config, err := OpenLoad(env)
	if err != nil {
		return nil, err
	}

	logger.InitLog(initLog(getAtomicLevel(config)), zap.AddCaller(), zap.AddCallerSkip(1))

	ops := []grpc.ServerOption{
		grpc.StreamInterceptor(
			grpcMiddleware.ChainStreamServer(
				logger.StreamConnectionInterceptor,
			),
		),
		grpc.UnaryInterceptor(
			grpcMiddleware.ChainUnaryServer(
				logger.ConnectionInterceptor,
			),
		), grpc.MaxConcurrentStreams(config.Server.MaxStreams),
		grpc.WriteBufferSize(config.Server.WriteBufferSize),
		grpc.ReadBufferSize(config.Server.ReadBufferSize),
		grpc.InitialWindowSize(config.Server.InitialConnWindowSize),
		grpc.InitialConnWindowSize(config.Server.InitialConnWindowSize),
		grpc.MaxHeaderListSize(config.Server.MaxHeaderListSize),
		grpc.MaxRecvMsgSize(config.Server.MaxRecvMsgSize),
	}

	config.Server.Opts = ops

	return config, nil

}

func cjaller(caller zapcore.EntryCaller, enc zapcore.PrimitiveArrayEncoder) {
	enc.AppendString(filepath.Base(caller.FullPath()))
}

func getAtomicLevel(config *Config) zap.AtomicLevel {
	var level zapcore.Level
	if err := level.Set(config.LogLevel); err != nil {
		log.Fatalf("failed to init log Level: %v", err)
	}

	return zap.NewAtomicLevelAt(level)
}

func initLog(level zap.AtomicLevel) zapcore.Core {
	stdout := zapcore.AddSync(os.Stdout)

	logDir := "./logs"
	if _, err := os.Stat(logDir); os.IsNotExist(err) {
		err := os.MkdirAll(logDir, 0755)
		if err != nil {
			panic(fmt.Sprintf("failed to create log directory: %v", err))
		}
	}

	logFilePath := logDir + "/app.json"

	file, err := os.OpenFile(logFilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		panic(fmt.Sprintf("failed to open log file: %v", err))
	}

	if fileInfo, err := file.Stat(); err == nil && fileInfo.Size() > 10<<20 {
		file.Truncate(0)
		file.Seek(0, 0)
	}

	fileSync := zapcore.AddSync(file)

	productionConfig := zap.NewProductionConfig()
	productionConfig.EncoderConfig.TimeKey = "time"
	productionConfig.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	productionConfig.EncoderConfig.CallerKey = "caller"
	productionConfig.EncoderConfig.EncodeCaller = cjaller
	productionConfig.EncoderConfig.EncodeLevel = zapcore.CapitalLevelEncoder

	devConfig := zap.NewDevelopmentConfig()
	devConfig.EncoderConfig.TimeKey = "time"
	devConfig.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	devConfig.EncoderConfig.CallerKey = "caller"
	devConfig.EncoderConfig.EncodeCaller = zapcore.ShortCallerEncoder
	devConfig.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder

	consoleEncoder := zapcore.NewConsoleEncoder(devConfig.EncoderConfig)
	fileEncoder := zapcore.NewJSONEncoder(productionConfig.EncoderConfig)

	return zapcore.NewTee(
		zapcore.NewCore(consoleEncoder, stdout, level),
		zapcore.NewCore(fileEncoder, fileSync, level),
	)

}
