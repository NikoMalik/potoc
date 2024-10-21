package logger

import (
	"context"
	"time"

	"github.com/NikoMalik/potoc/pkg/proto"
	"go.uber.org/zap"
	"google.golang.org/grpc"
)

func ConnectionInterceptor(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {

	start := time.Now()
	defer func() {
		Info("Client disconnected", zap.String("client", info.FullMethod), zap.Duration("duration", time.Since(start)))
	}()
	resp, err := handler(ctx, req)
	if err != nil {
		Error(err.Error(), zap.String("method", info.FullMethod), zap.Any("req", req))
		return nil, err
	}
	Info("Client connected", zap.String("method", info.FullMethod), zap.Any("req", req))

	return resp, err
}

func StreamConnectionInterceptor(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {

	start := time.Now()
	Info("Client connected", zap.String("method", info.FullMethod), zap.Time("time", start))

	wrapped := &wrappedServerStream{ServerStream: ss}

	err := handler(srv, wrapped)

	Info("Client disconnected", zap.String("method", info.FullMethod), zap.Duration("duration", time.Since(start)))

	return err
}

type wrappedServerStream struct {
	grpc.ServerStream
}

func (w *wrappedServerStream) SendMsg(m interface{}) error {
	err := w.ServerStream.SendMsg(m)

	if err == nil {
		switch msg := m.(type) {
		case []byte:
			if len(msg) > 100 {
				Debug("Sending message to client", zap.ByteString("msg", msg[:100]))
			} else {
				Debug("Sending message to client", zap.ByteString("msg", msg))
			}
		case *proto.DataResponse:
			if len(msg.Data) > 100 {
				Debug("Sending message to client", zap.ByteString("msg", msg.Data[:100]))
			} else {
				Debug("Sending message to client", zap.ByteString("msg", msg.Data))
			}
		default:
			Debug("Sending message to client", zap.Any("msg", msg))
		}
	}
	return w.ServerStream.SendMsg(m)
}

func (w *wrappedServerStream) RecvMsg(m interface{}) error {
	err := w.ServerStream.RecvMsg(m)

	if err == nil {
		switch msg := m.(type) {
		case []byte:
			if len(msg) > 100 {
				Debug("Received message from client", zap.ByteString("msg", msg[:100]))
			} else {
				Debug("Received message from client", zap.ByteString("msg", msg))
			}
		case *proto.DataRequest:
			if len(msg.EncodedData) > 100 {
				Debug("Received message from client", zap.ByteString("msg", msg.EncodedData[:100]))
			} else {
				Debug("Received message from client", zap.ByteString("msg", msg.EncodedData))
			}

		default:

			Debug("Received message from client", zap.Any("msg", msg))
		}
	}
	return err
}
