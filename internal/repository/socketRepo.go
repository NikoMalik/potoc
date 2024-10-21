package repository

import (
	"context"
	"errors"
	"sync"

	"github.com/NikoMalik/potoc/internal/logger"
	"github.com/NikoMalik/potoc/internal/models"
	"go.uber.org/zap"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var _ SocketRepo = (*socketRepo)(nil)

var socketDataPool = &sync.Pool{
	New: func() interface{} {
		return new(models.SocketData)
	},
}

type socketRepo struct {
	db *pgxpool.Pool
}

func NewSocketRepo(db *pgxpool.Pool) SocketRepo {
	return &socketRepo{db: db}
}

func (s *socketRepo) Create(ctx context.Context, data *models.SocketData) (string, error) {
	_, err := s.db.Query(ctx, "INSERT INTO socket_data (id, data) VALUES ($1, $2) RETURNING id", data.ID, data.Data)
	if err != nil {
		if errors.Is(err, context.Canceled) {
			return data.ID.String(), nil
		}
		logger.Error(err.Error())
		return "", err
	}
	return data.ID.String(), nil
}

func (s *socketRepo) Get(ctx context.Context, id string) (*models.SocketData, error) {
	var data = socketDataPool.Get().(*models.SocketData)
	err := s.db.QueryRow(ctx, "SELECT id, data FROM socket_data WHERE id = $1", id).Scan(&data.ID, &data.Data)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			logger.Warn("No rows found for ID", zap.String("id", id))
			socketDataPool.Put(data)

			return nil, nil
		}
		socketDataPool.Put(data)
		logger.Error(err.Error())
		return nil, err
	}
	socketDataPool.Put(data)

	return data, nil
}

func (s *socketRepo) Delete(ctx context.Context, id string) error {
	_, err := s.db.Exec(ctx, "DELETE FROM socket_data WHERE id = $1", id)
	if err != nil {
		logger.Error(err.Error())
		return err
	}
	return nil
}

func (s *socketRepo) DeleteAll(ctx context.Context) error {
	_, err := s.db.Exec(ctx, "DELETE FROM socket_data")
	if err != nil {
		logger.Error(err.Error())
		return err
	}
	return nil
}

func (s *socketRepo) Update(ctx context.Context, id string) (*models.SocketData, error) {
	var data = socketDataPool.Get().(*models.SocketData)
	err := s.db.QueryRow(ctx, "SELECT id, data FROM socket_data WHERE id = $1", id).Scan(&data.ID, &data.Data)
	if err != nil {
		socketDataPool.Put(data)
		logger.Error(err.Error())
		return nil, err
	}

	socketDataPool.Put(data)
	return data, nil
}

func (s *socketRepo) Count(ctx context.Context) (int, error) {
	var count int
	err := s.db.QueryRow(ctx, "SELECT COUNT(*) FROM socket_data").Scan(&count)
	if err != nil {
		logger.Error(err.Error())
		return 0, err
	}
	return count, nil
}
