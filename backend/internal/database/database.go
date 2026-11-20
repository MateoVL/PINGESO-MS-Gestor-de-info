package database

import (
	"context"
	"errors"
	"net/url"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/mateo/ms-gestor-de-info/backend/internal/config"
)

// ConnectDB crea un pool de conexiones que puede recuperar la conexión si PostgreSQL cae.
func ConnectDB(cfg *config.Config) (*pgxpool.Pool, error) {
	if (cfg.PostgresUser == "") != (cfg.PostgresPassword == "") {
		return nil, errors.New("POSTGRES_USER y POSTGRES_PASSWORD deben configurarse juntos")
	}

	connURI := cfg.PostgresURI
	u, err := url.Parse(connURI)
	if err != nil {
		return nil, err
	}
	if cfg.PostgresUser != "" {
		u.User = url.UserPassword(cfg.PostgresUser, cfg.PostgresPassword)
	}
	if (u.Path == "" || u.Path == "/") && cfg.PostgresDatabase != "" {
		u.Path = "/" + strings.TrimPrefix(cfg.PostgresDatabase, "/")
	}

	poolCfg, err := pgxpool.ParseConfig(u.String())
	if err != nil {
		return nil, err
	}
	return pgxpool.NewWithConfig(context.Background(), poolCfg)
}
