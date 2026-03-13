package database

import (
	"context"
	"fmt"
	"log"
	"shortlink/pkg/globals"
	sqlc "shortlink/sqlc/db"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

var DB *sqlc.Queries

func InitDB() error {
	connStr := fmt.Sprintf(
		"postgres://%s:%s@%s:%d/%s",
		globals.Config.Database.User,
		globals.Config.Database.Password,
		globals.Config.Database.Host,
		globals.Config.Database.Port,
		globals.Config.Database.DbName,
	)

	conf, err := pgxpool.ParseConfig(connStr)
	if err != nil {
		return fmt.Errorf("error parsing DB config: %v", err)
	}

	conf.MaxConns = 50
	conf.MinConns = 5
	conf.MaxConnLifetime = 30 * time.Minute
	conf.MaxConnIdleTime = 5 * time.Minute
	conf.HealthCheckPeriod = 1 * time.Minute

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	DBPool, err := pgxpool.NewWithConfig(ctx, conf)
	if err != nil {
		return fmt.Errorf("error creating DB pool: %v", err)
	}

	DB = sqlc.New(DBPool)

	if err := DBPool.Ping(ctx); err != nil {
		return fmt.Errorf("db ping error: %v", err)
	}

	log.Println("Connected")

	return nil
}
