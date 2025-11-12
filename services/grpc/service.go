package grpc

import (
	"context"
	"fmt"
	"net"
	"sync/atomic"

	"github.com/Sandwichzzy/gas-oracle/database"
	"github.com/Sandwichzzy/gas-oracle/services/grpc/gasFeePb"
	"github.com/ethereum/go-ethereum/log"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

const MaxRecvMessageSize = 1024 * 1024 * 30000

type TokenPriceRpcConfig struct {
	Host string
	Port int
}

type TokenPriceRpcService struct {
	*TokenPriceRpcConfig

	db *database.DB

	gasFeePb.UnimplementedTokenGasPriceServicesServer
	stopped atomic.Bool
}

func NewTokenPriceRpcService(conf *TokenPriceRpcConfig, db *database.DB) (*TokenPriceRpcService, error) {
	return &TokenPriceRpcService{
		TokenPriceRpcConfig: conf,
		db:                  db,
	}, nil
}

func (ms *TokenPriceRpcService) Start(ctx context.Context) error {
	go func(ms *TokenPriceRpcService) {
		rpcAddr := fmt.Sprintf("%s:%d", ms.TokenPriceRpcConfig.Host, ms.TokenPriceRpcConfig.Port)
		listener, err := net.Listen("tcp", rpcAddr)
		if err != nil {
			log.Error("Could not start tcp listener. ")
		}
		opt := grpc.MaxRecvMsgSize(MaxRecvMessageSize)
		gs := grpc.NewServer(
			opt,
			grpc.ChainUnaryInterceptor(
				nil,
			),
		)
		reflection.Register(gs)
		gasFeePb.RegisterTokenGasPriceServicesServer(gs, ms)
		log.Info("grpc info", "addr", listener.Addr())
		if err := gs.Serve(listener); err != nil {
			log.Error("start rpc server fail", "err", err)
		}
	}(ms)
	return nil
}

func (ms *TokenPriceRpcService) Stop(ctx context.Context) error {
	ms.stopped.Store(true)
	return nil
}

func (ms *TokenPriceRpcService) Stopped() bool {
	return ms.stopped.Load()
}
