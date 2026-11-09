package database

import (
	"errors"

	"github.com/mateo/ms-gestor-de-info/backend/internal/config"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// ConnectDB crea un cliente que puede recuperar la conexión si MongoDB cae.
func ConnectDB(cfg *config.Config) (*mongo.Client, error) {
	if (cfg.MongoUser == "") != (cfg.MongoPassword == "") {
		return nil, errors.New("MONGO_USER y MONGO_PASSWORD deben configurarse juntos")
	}
	opts := options.Client().ApplyURI(cfg.MongoURI)
	if cfg.MongoUser != "" {
		opts.SetAuth(options.Credential{
			AuthSource: cfg.MongoDatabase,
			Username:   cfg.MongoUser,
			Password:   cfg.MongoPassword,
		})
	}
	return mongo.Connect(opts)
}
