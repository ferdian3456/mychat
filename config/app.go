package config

import (
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/julienschmidt/httprouter"
	"github.com/knadh/koanf/v2"
	zapLog "go.uber.org/zap"
)

type ServerConfig struct {
	Router *httprouter.Router
	DB     *pgxpool.Pool
	Log    *zapLog.Logger
	Config *koanf.Koanf
}
