package grpc

import (
	"gas-oracle-main/database"
	"sync/atomic"

	"gas-oracle-main/proto/gasfee"
)

const MaxRecvMessageSize = 1024 * 1024 * 30000

type TokenPriceRpcConfig struct {
	Host string
	Port int
}

type TokenPriceRpcService struct {
	*TokenPriceRpcConfig

	db *database.DB

	gasfee.UnimplementedTokenGasPriceServicesServer
	stopped atomic.Bool
}
