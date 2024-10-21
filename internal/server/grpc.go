package server

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"net"
	"strings"

	lowlevelfunctions "github.com/NikoMalik/low-level-functions"
	"github.com/NikoMalik/potoc/internal/config"
	"github.com/NikoMalik/potoc/internal/logger"
	"github.com/NikoMalik/potoc/internal/models"
	"github.com/NikoMalik/potoc/internal/repository"
	"github.com/NikoMalik/potoc/pkg/proto"
	"github.com/NikoMalik/uuid"
	"google.golang.org/grpc"
)

var (
	_errCancelContextConn = errors.New("rpc error: code = Canceled desc = context canceled")
	_errServerNotInit     = errors.New("server not init")
)

var _ proto.DataTranferServer = (*dataTransferServer)(nil)

type GRPC struct {
	grpc               *grpc.Server
	dataTransferServer *dataTransferServer
}

func NewGRPC(config *config.Config, repo *repository.Repositories) *GRPC {
	grpc := grpc.NewServer(config.Server.Opts...)

	dataTrans := NewTransfer(repo.SocketRepo)

	proto.RegisterDataTranferServer(grpc, dataTrans)

	return &GRPC{
		grpc:               grpc,
		dataTransferServer: dataTrans,
	}
}

func (s *GRPC) Run(ln net.Listener) error {
	if s.grpc == nil {
		return _errServerNotInit
	}
	return s.grpc.Serve(ln)
}

func (s *GRPC) Stop() {
	s.grpc.GracefulStop()
}

func (s *GRPC) PanicStop() {
	s.grpc.Stop()
}

type dataTransferServer struct {
	proto.UnimplementedDataTranferServer
	repo repository.SocketRepo
}

func NewTransfer(repo repository.SocketRepo) *dataTransferServer {
	return &dataTransferServer{
		repo: repo,
	}
}

func (d *dataTransferServer) GetData(stream proto.DataTranfer_GetDataServer) error {

	dataChannel := make(chan *models.SocketData)
	errChannel := make(chan error, 1)

	go func() {
		for {
			req, err := stream.Recv()
			if err == io.EOF {
				close(dataChannel)
				errChannel <- nil
				return
			}

			if err != nil {
				errChannel <- err
				return
			}
			decodedData, err := base64.StdEncoding.DecodeString(lowlevelfunctions.String(req.GetEncodedData()))
			if err != nil {
				errChannel <- fmt.Errorf("Faildef to decode base64: " + err.Error() + ", Input:" + lowlevelfunctions.String(req.GetEncodedData()))
				return
			}

			socketData := &models.SocketData{
				ID:   uuid.New(),
				Data: decodedData,
			}

			dataChannel <- socketData

		}
	}()

	go func() {

		for socketData := range dataChannel {

			_, err := d.repo.Create(stream.Context(), socketData)
			if err != nil {
				errChannel <- err
				return
			}

			logger.Debug("Data received and saved with ID: " + socketData.ID.String())
			if err = stream.Send(&proto.DataResponse{
				Status: "ok",
				Msg:    "Data received and saved",
				Data:   lowlevelfunctions.StringToBytes(socketData.ID.String()),
			}); err != nil {
				errChannel <- err
				return
			}

		}
	}()

	if err := <-errChannel; err != nil {
		if errors.Is(err, context.Canceled) || strings.Contains(err.Error(), _errCancelContextConn.Error()) {
			logger.Info("Client calcel connection")
			return nil
		}
		logger.Error(err.Error())
		return err
	}

	return nil
}

func (d *dataTransferServer) FetchData(stream proto.DataTranfer_FetchDataServer) error {
	dataChannel := make(chan *models.SocketData)
	errChannel := make(chan error, 1)

	go func() {
		for {
			req, err := stream.Recv()
			if err == io.EOF {
				close(dataChannel)
				errChannel <- nil
				return
			}
			if err != nil {
				errChannel <- err
				return
			}

			socketID := req.GetSocketId()
			if err := stream.Context().Err(); err != nil {
				logger.Info("Client disconnected before fetching data")
				return
			}
			if socketID == "" {
				errChannel <- errors.New("empty SocketId")
				return
			}
			socketData, err := d.repo.Get(stream.Context(), socketID)
			if err != nil {
				errChannel <- fmt.Errorf("Error fetching data for ID:" + socketID)
				return
			}

			dataChannel <- socketData
		}
	}()

	go func() {

		for socketData := range dataChannel {
			encodedData := base64.StdEncoding.EncodeToString(socketData.Data)

			err := stream.Send(&proto.DataResponse{
				Status: "ok",
				Msg:    "success fetched",
				Data:   []byte(encodedData),
			})
			if err != nil {
				errChannel <- err
				return
			}

			logger.Debug("Data sent to client with ID: " + socketData.ID.String())
		}
	}()

	if err := <-errChannel; err != nil {
		if errors.Is(err, context.Canceled) || strings.Contains(err.Error(), _errCancelContextConn.Error()) {
			logger.Info("Client calcel connection")
			return nil
		}
		logger.Error(err.Error())
		return err
	}

	return nil
}
