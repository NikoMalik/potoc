package repository

import (
	"context"

	"github.com/NikoMalik/potoc/internal/models"
	"github.com/jackc/pgx/v5/pgxpool"
)

type SocketRepo interface {
	Create(context.Context, *models.SocketData) (string, error)
	Get(context.Context, string) (*models.SocketData, error)
	Delete(context.Context, string) error
	DeleteAll(context.Context) error
	Count(context.Context) (int, error)
	Update(context.Context, string) (*models.SocketData, error)
}

type RandomRepo interface {
	GenerateRandomData(context.Context) error
	CheckIfExists(context.Context) (bool, error)
}

type Repositories struct {
	SocketRepo SocketRepo
	RandomRepo RandomRepo
}

func NewRepositories(db *pgxpool.Pool) *Repositories {
	return &Repositories{
		SocketRepo: NewSocketRepo(db),
		RandomRepo: NewRandomRepo(db),
	}
}
