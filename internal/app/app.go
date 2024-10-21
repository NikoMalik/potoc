package app

import (
	"context"
	"errors"
	"log"
	"os"

	"github.com/NikoMalik/potoc/internal/config"
	"github.com/NikoMalik/potoc/internal/database"
	"github.com/NikoMalik/potoc/internal/logger"
	"github.com/NikoMalik/potoc/internal/repository"
	"github.com/NikoMalik/potoc/internal/server"
	"github.com/subosito/gotenv"
	"go.uber.org/zap"
)

var (
	_errorInitial = errors.New("failed to initialize app")
)

type App struct {
	Config *config.Config
	DB     *repository.Repositories
	Server *server.Server
}

//	func cjaller(caller zapcore.EntryCaller, enc zapcore.PrimitiveArrayEncoder) {
//		enc.AppendString(filepath.Base(caller.FullPath()))
//	}
//
//	func getAtomicLevel(config *config.Config) zap.AtomicLevel {
//		var level zapcore.Level
//		if err := level.Set(config.LogLevel); err != nil {
//			log.Fatalf("failed to init log Level: %v", err)
//		}
//
//		return zap.NewAtomicLevelAt(level)
//	}
//
//	func initLog(level zap.AtomicLevel) zapcore.Core {
//		stdout := zapcore.AddSync(os.Stdout)
//
//		logDir := "./logs"
//		if _, err := os.Stat(logDir); os.IsNotExist(err) {
//			err := os.MkdirAll(logDir, 0755)
//			if err != nil {
//				panic(fmt.Sprintf("failed to create log directory: %v", err))
//			}
//		}
//
//		logFilePath := logDir + "/app.log"
//
//		file, err := os.OpenFile(logFilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
//		if err != nil {
//			panic(fmt.Sprintf("failed to open log file: %v", err))
//		}
//
//		if fileInfo, err := file.Stat(); err == nil && fileInfo.Size() > 10000 {
//			file.Truncate(0)
//			file.Seek(0, 0)
//		}
//
//		fileSync := zapcore.AddSync(file)
//
//		productionConfig := zap.NewProductionConfig()
//		productionConfig.EncoderConfig.TimeKey = "time"
//		productionConfig.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
//		productionConfig.EncoderConfig.CallerKey = "caller"
//		productionConfig.EncoderConfig.EncodeCaller = cjaller
//		productionConfig.EncoderConfig.EncodeLevel = zapcore.CapitalLevelEncoder
//
//		devConfig := zap.NewDevelopmentConfig()
//		devConfig.EncoderConfig.TimeKey = "time"
//		devConfig.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
//		devConfig.EncoderConfig.CallerKey = "caller"
//		devConfig.EncoderConfig.EncodeCaller = zapcore.ShortCallerEncoder
//		devConfig.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
//
//		consoleEncoder := zapcore.NewConsoleEncoder(devConfig.EncoderConfig)
//		fileEncoder := zapcore.NewJSONEncoder(productionConfig.EncoderConfig)
//
//		return zapcore.NewTee(
//			zapcore.NewCore(consoleEncoder, stdout, level),
//			zapcore.NewCore(fileEncoder, fileSync, level),
//		)
//
// }
func NewApp(ctx context.Context) (*App, error) {
	err := gotenv.Load()
	if err != nil {
		panic(err)
	}
	env := getEnv()
	config, err := config.NewConfig(env)
	if err != nil {
		log.Fatal(err)
	}
	// logger.InitLog(initLog(getAtomicLevel(config)), zap.AddCaller(), zap.AddCallerSkip(1))

	logger.Info("Init server", zap.String("env", env))

	db := database.NewDB()

	repos := repository.NewRepositories(db)
	go func() {
		err := repos.RandomRepo.GenerateRandomData(ctx)
		if err != nil {
			logger.Error("Failed to generate random data", zap.Error(err))
		}
	}()
	server := server.NewServer(config, repos)

	app := &App{
		Config: config,
		DB:     repos,
		Server: server,
	}

	return app, nil
}

func getEnv() string {
	env := os.Getenv("ENVIRONMENT")
	if env == "" {
		env = "local"
		log.Println("Running in local mode")

	}
	switch env {
	case "dev":
		log.Println("Running in development mode")
	case "prod":
		log.Println("Running in production mode")
	}

	return env

}

func (app *App) Run() error {
	if app == nil {
		return _errorInitial
	}

	return app.Server.Run()
}

func (app *App) Close(ctx context.Context) error {
	if app == nil {
		return _errorInitial
	}
	done := make(chan error, 1)
	go func() {
		done <- app.Server.Stop()
	}()
	select {
	case <-ctx.Done():
		logger.Warn("Shutdown timed out, start panic stop")
		app.Server.PanicStop()
		return ctx.Err()
	case err := <-done:
		if err != nil {
			logger.Error("ERROR DURING SERVER SHUTDOWN", zap.Error(err))
		}
		logger.Info("Server shutdown gracefully")
		return nil
	}
}
