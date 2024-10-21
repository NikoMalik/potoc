package repository

import (
	"context"
	"math/rand"
	"time"

	"strconv"
	"sync"

	"github.com/NikoMalik/potoc/internal/logger"
	"github.com/NikoMalik/potoc/internal/models"
	"github.com/jackc/pgx/v5/pgxpool"
)

type randomRepo struct {
	db   *pgxpool.Pool
	once sync.Once
}

func NewRandomRepo(db *pgxpool.Pool) RandomRepo {
	return &randomRepo{
		db: db,
	}
}

func (r *randomRepo) GenerateRandomData(ctx context.Context) error {
	var err error
	r.once.Do(func() {
		exists, err := r.CheckIfExists(ctx)
		if err != nil {
			logger.Error(err.Error())
			return
		}

		if exists {
			logger.Info("Random data already exists")
			return
		}
		logger.Info("Generating random data...")
		for i := 0; i < 5000; i++ {
			go func(i int) {

				randomData := &models.RandomData{
					Name:        "Name_" + strconv.Itoa(i),
					Description: "Description_" + strconv.Itoa(i),
					Value:       rand.Intn(5000),
					CreatedAt:   time.Now(),
				}

				_, err := r.db.Exec(ctx, "INSERT INTO random_data (name, description, value, created_at) VALUES ($1, $2, $3, $4)",
					randomData.Name, randomData.Description, randomData.Value, randomData.CreatedAt)
				if err != nil {
					logger.Error(err.Error())
					return
				}
			}(i)

		}
	})
	return err
}

func (r *randomRepo) CheckIfExists(ctx context.Context) (bool, error) {
	var count int
	err := r.db.QueryRow(ctx, "SELECT COUNT(*) FROM random_data").Scan(&count)
	if err != nil {
		logger.Error(err.Error())
		return false, err
	}
	logger.Info("Count: " + strconv.Itoa(count))
	return count > 0, nil
}
