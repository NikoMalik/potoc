package server

import (
	"fmt"
	"net"

	"github.com/NikoMalik/potoc/internal/config"
	"github.com/NikoMalik/potoc/internal/logger"
	"github.com/NikoMalik/potoc/internal/repository"
)

type Server struct {
	config *config.Config
	grpc   *GRPC
}

func NewServer(config *config.Config, repo *repository.Repositories) *Server {
	return &Server{
		config: config,
		grpc:   NewGRPC(config, repo),
	}
}

func (s *Server) Run() error {
	ln, err := net.Listen("tcp", net.JoinHostPort(s.config.Server.Host, s.config.Server.Port))
	if err != nil {
		logger.Fatal(fmt.Sprintf("failed to listen: %v", err))
		return err
	}
	r := s.grpc.Run(ln)
	if r != nil {
		logger.Fatal(fmt.Sprintf("failed to run: %v", err))
		return err
	}
	return r
}

func (s *Server) Stop() error {
	s.grpc.Stop()
	return nil
}

func (s *Server) PanicStop() {
	s.grpc.PanicStop()

}
