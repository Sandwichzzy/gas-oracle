package synchronizer

import (
	"context"
	"fmt"
	"math/big"
	"sync/atomic"
	"time"

	"github.com/Sandwichzzy/gas-oracle/common/tasks"
	"github.com/Sandwichzzy/gas-oracle/database"
	"github.com/Sandwichzzy/gas-oracle/synchronizer/node"
	"github.com/ethereum/go-ethereum/log"
	"github.com/google/uuid"
)

type OracleSynchronizer struct {
	loopInterval   time.Duration
	db             *database.DB
	ethClient      node.EthClient
	blockOffset    uint64
	chainId        uint64
	nativeToken    string
	decimal        uint8
	stopped        atomic.Bool
	resourceCtx    context.Context
	resourceCancel context.CancelFunc
	tasks          tasks.Group
}

func NewOracleSynchronizer(db *database.DB, client node.EthClient, blockOffset uint64, chainId uint64, nativeToken string, decimal uint8, loopInterval time.Duration, shutdown context.CancelCauseFunc) (*OracleSynchronizer, error) {
	resCtx, resCancel := context.WithCancel(context.Background())

	return &OracleSynchronizer{
		loopInterval: loopInterval,
		db:           db,
		chainId:      chainId,
		nativeToken:  nativeToken,
		decimal:      decimal,
		blockOffset:  blockOffset,
		ethClient:    client,
		tasks: tasks.Group{HandleCrit: func(err error) {
			shutdown(fmt.Errorf("critical error in selaginella processor: %w", err))
		}},
		resourceCancel: resCancel,
		resourceCtx:    resCtx,
	}, nil
}
func (os *OracleSynchronizer) Stop(ctx context.Context) error {
	os.stopped.Store(true)
	return nil
}

func (os *OracleSynchronizer) Stopped() bool {
	return os.stopped.Load()
}

func (os *OracleSynchronizer) Start(ctx context.Context) error {
	l1FeeTicker := time.NewTicker(os.loopInterval)
	os.tasks.Go(func() error {
		for range l1FeeTicker.C {
			fee, err := os.processTokenPrice(os.chainId)
			if err != nil {
				log.Error("process token price error", "err", err)
				log.Error(err.Error())
			}
			log.Info("get gas fee", "fee", fee, "chainId", os.chainId)
			gasFee := &database.GasFee{
				GUID:       uuid.New(),
				ChainId:    big.NewInt(int64(os.chainId)),
				Decimal:    os.decimal,
				TokenName:  os.nativeToken,
				PredictFee: fee.String(),
				Timestamp:  uint64(time.Now().Unix()),
			}
			err = os.db.GasFee.StoreOrUpdateGasFee(gasFee)
			if err != nil {
				log.Error("Oracle synchronizer store or update gas fee fail", "err", err)
				return err
			}
		}
		return nil
	})
	return nil
}

func (os *OracleSynchronizer) processTokenPrice(chainId uint64) (*big.Int, error) {
	log.Info("process token price", "chainId", chainId)
	latestBlockN, err := os.ethClient.GetLatestBlock(context.Background())
	if err != nil {
		log.Error("failed to get l1 latest block", "err", err)
		return nil, err
	}
	totalAllBlocksFee := big.NewInt(0) // 所有区块的总费用
	totalTransactions := 0             // 所有交易数量
	log.Info("start handle block fee", "blockOffset", os.blockOffset, "latestBlockN", latestBlockN.String(), "chainId", chainId)
	for i := 0; i < int(os.blockOffset); i++ {
		blockNumber := int(latestBlockN.Uint64()) - i // 从最新区块向前追溯
		txs, _, err := os.ethClient.BlockDetailByNumber(context.Background(), big.NewInt(int64(blockNumber)))
		if err != nil {
			log.Error("failed to get block", "blockNum", blockNumber, "err", err)
			return nil, err
		}
		log.Info("successfully get block info", "block_num", blockNumber, "tx_len", len(txs))
		if len(txs) == 0 {
			continue
		}
		blockReceipts, err := os.ethClient.BlockReceiptsByNumber(context.Background(), big.NewInt(int64(blockNumber)))
		if err != nil {
			log.Error("failed to get blockreceipts", "blockNum", blockNumber, "err", err)
			return nil, err
		}
		// 计算单个区块的总gas费用
		blockTotalFee := big.NewInt(0)
		for _, receipt := range blockReceipts {
			if receipt == nil {
				continue
			}
			// effectiveGasPrice 表示交易实际生效的每单位 Gas 价格（以 wei 为单位）
			transactionFee := new(big.Int).Mul(
				receipt.EffectiveGasPrice,
				new(big.Int).SetUint64(receipt.GasUsed),
			)
			blockTotalFee.Add(blockTotalFee, transactionFee)
		}
		// 累加到全局统计
		totalAllBlocksFee.Add(totalAllBlocksFee, blockTotalFee)
		totalTransactions += len(txs)
		log.Info("block processed", "block", blockNumber, "blockFee", blockTotalFee, "txs", len(txs))
	}

	// 计算真正的平均交易费用
	if totalTransactions == 0 {
		return big.NewInt(0), nil
	}
	//平均交易费用 = 所有区块的总Gas费用 ÷ 所有交易数量
	averageFee := new(big.Int).Div(totalAllBlocksFee, big.NewInt(int64(totalTransactions)))
	log.Info("successfully get estimated fee", "chainId", chainId, "fee", averageFee, "totalTxs", totalTransactions)
	return averageFee, nil
}
