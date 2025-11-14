package gas_oracle

import (
	"context"
	"errors"
	"fmt"
	"sync/atomic"
	"time"

	"github.com/Sandwichzzy/gas-oracle/config"
	"github.com/Sandwichzzy/gas-oracle/database"
	"github.com/Sandwichzzy/gas-oracle/synchronizer"
	"github.com/Sandwichzzy/gas-oracle/synchronizer/node"
	"github.com/Sandwichzzy/gas-oracle/worker"
	"github.com/ethereum/go-ethereum/log"
)

type GasOracle struct {
	db           *database.DB
	ethClient    map[uint64]node.EthClient
	synchronizer map[uint64]*synchronizer.OracleSynchronizer
	workerHandle *worker.WorkerHandle
	symbolList   []string
	shutdown     context.CancelCauseFunc
	stopped      atomic.Bool
	backOffset   uint64
	loopInternal time.Duration
	chainIdList  []uint64
}

func NewGasOracle(ctx context.Context, cfg *config.Config, shutdown context.CancelCauseFunc) (*GasOracle, error) {
	log.Info("new gas oracle start️")
	out := &GasOracle{
		loopInternal: cfg.LoopInternal,
		backOffset:   cfg.BackOffset,
		shutdown:     shutdown,
	}
	if err := out.initFromConfig(ctx, cfg); err != nil {
		return nil, errors.Join(err, out.Stop(ctx))
	}
	log.Info("new gas oracle success")
	return out, nil
}

func (as *GasOracle) Start(ctx context.Context) error {
	for i := range as.chainIdList {
		log.Info("starting sync", "chainId", as.chainIdList[i])
		realChainId := as.chainIdList[i]
		if err := as.synchronizer[realChainId].Start(context.Background()); err != nil {
			return fmt.Errorf("failed to start chain sync: %w", err)
		}
	}

	err := as.workerHandle.Start()
	if err != nil {
		log.Error("start work handle fail", "err", err)
		return err
	}

	return nil
}

func (as *GasOracle) Stop(ctx context.Context) error {
	var result error
	for i := range as.chainIdList {
		if as.synchronizer[as.chainIdList[i]] != nil {
			if err := as.synchronizer[as.chainIdList[i]].Stop(ctx); err != nil {
				result = errors.Join(result, fmt.Errorf("failed to close synchronizer: %w", err))
			}
		}
		if as.ethClient[as.chainIdList[i]] != nil {
			as.ethClient[as.chainIdList[i]].Close()
		}
	}

	if as.db != nil {
		if err := as.db.Close(); err != nil {
			result = errors.Join(result, fmt.Errorf("failed to close DB: %w", err))
		}
	}

	err := as.workerHandle.Close()
	if err != nil {
		log.Error("close work handle fail", "err", err)
		return err
	}

	as.stopped.Store(true)

	log.Info("gas oracle stopped")

	return result
}

func (as *GasOracle) Stopped() bool {
	return as.stopped.Load()
}

func (as *GasOracle) initFromConfig(ctx context.Context, cfg *config.Config) error {
	if err := as.initRPCClients(ctx, cfg); err != nil {
		return fmt.Errorf("failed to start RPC clients: %w", err)
	}

	if err := as.initDB(ctx, cfg.MasterDb); err != nil {
		return fmt.Errorf("failed to init DB: %w", err)
	}

	if err := as.initSynchronizer(cfg); err != nil {
		return fmt.Errorf("failed to init Sync: %w", err)
	}

	if err := as.initWorkerHandle(cfg); err != nil {
		return fmt.Errorf("failed to init work handle: %w", err)
	}

	return nil
}

func (as *GasOracle) initRPCClients(ctx context.Context, conf *config.Config) error {
	for i := range conf.RPCs {
		log.Info("Init rpc client", "ChainId", conf.RPCs[i].ChainId, "RpcUrl", conf.RPCs[i].RpcUrl)
		rpc := conf.RPCs[i]
		ethClient, err := node.DialEthClient(ctx, rpc.RpcUrl)
		if err != nil {
			log.Error("dial eth client fail", "err", err)
			return fmt.Errorf("failed to dial L1 client: %w", err)
		}
		if as.ethClient == nil {
			as.ethClient = make(map[uint64]node.EthClient)
		}
		as.ethClient[rpc.ChainId] = ethClient
		as.chainIdList = append(as.chainIdList, rpc.ChainId)
	}
	log.Info("Init rpc client success")
	return nil
}

func (as *GasOracle) initDB(ctx context.Context, cfg config.Database) error {
	db, err := database.NewDB(ctx, cfg)
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}
	as.db = db
	log.Info("Init database success")
	return nil
}

func (as *GasOracle) initSynchronizer(config *config.Config) error {
	for i := range config.RPCs {
		log.Info("Init synchronizer", "chainId", config.RPCs[i].ChainId)
		rpcItem := config.RPCs[i]

		synchronizerTemp, err := synchronizer.NewOracleSynchronizer(as.db, as.ethClient[config.RPCs[i].ChainId], as.backOffset, rpcItem.ChainId, rpcItem.NativeToken, rpcItem.Decimal, as.loopInternal, as.shutdown)
		if err != nil {
			log.Error("new oracle synchronizer fail", "err", err)
			return err
		}
		if as.synchronizer == nil {
			as.synchronizer = make(map[uint64]*synchronizer.OracleSynchronizer)
		}
		as.synchronizer[rpcItem.ChainId] = synchronizerTemp
	}
	return nil
}

func (as *GasOracle) initWorkerHandle(config *config.Config) error {
	var symbolList []worker.Symbols
	for _, symbol := range config.Symbols {
		item := worker.Symbols{
			Name:    symbol.Name,
			Decimal: symbol.Decimal,
		}
		symbolList = append(symbolList, item)
	}
	wConf := &worker.WorkerHandleConfig{
		BaseUrl:      config.SkyeyeUrl,
		LoopInterval: time.Second * 5,
		SymbolList:   symbolList,
	}
	handle, err := worker.NewWorkerHandle(as.db, wConf, as.shutdown)
	if err != nil {
		log.Error("new work handle fail", "err", err)
		return err
	}
	as.workerHandle = handle
	return nil
}
