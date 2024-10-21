package database

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	_ "github.com/lib/pq"
	"github.com/spf13/viper"
)

func NewDB() *pgxpool.Pool {
	log.Println("connecting to database")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	db, err := pgxpool.New(ctx, getDSN())
	defer cancel()
	if err != nil {
		log.Fatal(err)

	}
	err = db.Ping(ctx)
	if err != nil {
		log.Panic(err)

	}

	dr := stdlib.OpenDBFromPool(db)

	driver, err := postgres.WithInstance(dr, &postgres.Config{})
	if err != nil {
		log.Fatal(err)

	}

	m, err := migrate.NewWithDatabaseInstance(
		"file://internal/migrations/",
		"postgres",
		driver,
	)
	if err != nil {
		log.Panic(err)

	}

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		log.Panic(err)

	}

	version, dirty, _ := m.Version()
	log.Printf("Loaded migrations: version=%d dirty=%t", version, dirty)
	log.Printf("Dirty: %t", dirty)

	log.Println("Database successfully connected")

	return db
}

func getDSN() string {
	return fmt.Sprintf("host=%s port=%d user=%s "+
		"password=%s dbname=%s sslmode=%s ",
		viper.GetString("DB_HOST"),
		viper.GetInt("DB_PORT"),
		viper.GetString("DB_USER"),
		viper.GetString("DB_PASSWORD"),
		viper.GetString("DB_NAME"),
		viper.GetString("DB_SSLMODE"),
	)
}
