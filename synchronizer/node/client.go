package node

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"math/big"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rpc"

	"github.com/Sandwichzzy/gas-oracle/synchronizer/retry"
)

const (
	defaultDialTimeout     = 5 * time.Second
	defaultDialAttempts    = 5
	defaultRequestTimeout  = 10 * time.Second
	defaultWaitTransaction = 5 * time.Minute
)

type EthClient interface {
	GetLatestBlock(ctx context.Context) (*big.Int, error)
	TxReceiptDetailByHash(ctx context.Context, hash common.Hash) (*types.Receipt, error)
	BlockReceiptsByNumber(ctx context.Context, number *big.Int) ([]*types.Receipt, error)
	HeaderByNumber(ctx context.Context, number *big.Int) (*types.Header, error)
	TxByHash(ctx context.Context, hash common.Hash) (*types.Transaction, error)
	BlockDetailByNumber(ctx context.Context, number *big.Int) ([]string, *big.Int, error)
	Close()
}

type clnt struct {
	rpc RPC
}

func DialEthClient(ctx context.Context, rpcUrl string) (EthClient, error) {
	ctx, cancel := context.WithTimeout(ctx, defaultDialTimeout)
	defer cancel()

	bOff := retry.Exponential()
	rpcClient, err := retry.Do(ctx, defaultDialAttempts, bOff, func() (*rpc.Client, error) {
		if !IsURLAvailable(rpcUrl) {
			return nil, fmt.Errorf("address unavailable (%s)", rpcUrl)
		}

		client, err := rpc.DialContext(ctx, rpcUrl)
		if err != nil {
			return nil, fmt.Errorf("failed to dial address (%s): %w", rpcUrl, err)
		}

		return client, nil
	})

	if err != nil {
		return nil, err
	}

	return &clnt{rpc: NewRPC(rpcClient)}, nil
}

func (c *clnt) SendTransaction(ctx context.Context, tx *types.Transaction) error {
	data, err := tx.MarshalBinary()
	if err != nil {
		return err
	}
	return c.rpc.CallContext(ctx, nil, "eth_sendRawTransaction", hexutil.Encode(data))
}

func (c *clnt) TxReceiptDetailByHash(ctx context.Context, hash common.Hash) (*types.Receipt, error) {
	ctxwt, cancel := context.WithTimeout(context.Background(), defaultWaitTransaction)
	defer cancel()
	var txReceipt *types.Receipt
	err := c.rpc.CallContext(ctxwt, &txReceipt, "eth_getTransactionReceipt", hash)
	if err != nil {
		return nil, err
	} else if txReceipt == nil {
		return nil, ethereum.NotFound
	}
	return txReceipt, nil
}

func (c *clnt) BlockReceiptsByNumber(ctx context.Context, number *big.Int) ([]*types.Receipt, error) {
	ctxwt, cancel := context.WithTimeout(context.Background(), defaultWaitTransaction)
	defer cancel()
	var blockReceipts []*types.Receipt
	err := c.rpc.CallContext(ctxwt, &blockReceipts, "eth_getBlockReceipts", toBlockNumArg(number))
	if err != nil {
		return nil, err
	} else if blockReceipts == nil {
		return nil, ethereum.NotFound
	}
	return blockReceipts, nil
}

func (c *clnt) GetLatestBlock(ctx context.Context) (*big.Int, error) {
	ctxwt, cancel := context.WithTimeout(ctx, defaultRequestTimeout)
	defer cancel()

	blockHeight, err := c.BlockNumber(ctxwt)
	if err != nil {
		log.Error("cannot retrieve the current chain ID", "err", err)
		return nil, err
	}
	return big.NewInt(int64(blockHeight)), nil
}

func (c *clnt) BlockNumber(ctx context.Context) (uint64, error) {
	var result hexutil.Uint64
	err := c.rpc.CallContext(ctx, &result, "eth_blockNumber")
	return uint64(result), err
}

func (c *clnt) HeaderByNumber(ctx context.Context, number *big.Int) (*types.Header, error) {
	var head *types.Header
	err := c.rpc.CallContext(ctx, &head, "eth_getBlockByNumber", toBlockNumArg(number), false)
	if err == nil && head == nil {
		err = ethereum.NotFound
	}
	return head, err
}

func (c *clnt) TxByHash(ctx context.Context, hash common.Hash) (*types.Transaction, error) {
	ctxwt, cancel := context.WithTimeout(ctx, defaultRequestTimeout)
	defer cancel()

	var tx *types.Transaction
	err := c.rpc.CallContext(ctxwt, &tx, "eth_getTransactionByHash", hash)
	if err != nil {
		return nil, err
	} else if tx == nil {
		return nil, ethereum.NotFound
	}

	return tx, nil
}

type rpcBlock struct {
	Hash         common.Hash `json:"hash"`
	Transactions []string    `json:"transactions"`
	BaseFee      string      `json:"baseFeePerGas"`
}

func (c *clnt) BlockDetailByNumber(ctx context.Context, number *big.Int) ([]string, *big.Int, error) {
	ctxwt, cancel := context.WithTimeout(ctx, defaultRequestTimeout)
	defer cancel()

	var block rpcBlock
	var BaseFeeB *big.Int
	err := c.rpc.CallContext(ctxwt, &block, "eth_getBlockByNumber", toBlockNumArg(number), false)
	if err != nil {
		log.Error("Call eth_getBlockByNumber method fail", "err", err)
		return nil, nil, err
	} else if &block == nil {
		log.Warn("block not found")
		return nil, nil, ethereum.NotFound
	}
	if block.BaseFee == "" {
		BaseFeeB = big.NewInt(0)
	} else {
		BaseFeeB, _ = new(big.Int).SetString(block.BaseFee[2:], 16)
	}

	return block.Transactions, BaseFeeB, nil
}

func (c *clnt) CallContract(ctx context.Context, msg ethereum.CallMsg, blockNumber *big.Int) ([]byte, error) {
	var hex hexutil.Bytes
	err := c.rpc.CallContext(ctx, &hex, "eth_call", toCallArg(msg), toBlockNumArg(blockNumber))
	if err != nil {
		return nil, err
	}
	return hex, nil
}

func (c *clnt) Close() {
	c.rpc.Close()
}

type RPC interface {
	Close()
	CallContext(ctx context.Context, result any, method string, args ...any) error
	BatchCallContext(ctx context.Context, b []rpc.BatchElem) error
}

type rpcClient struct {
	rpc *rpc.Client
}

func NewRPC(client *rpc.Client) RPC {
	return &rpcClient{client}
}

func (c *rpcClient) Close() {
	c.rpc.Close()
}

func (c *rpcClient) CallContext(ctx context.Context, result any, method string, args ...any) error {
	err := c.rpc.CallContext(ctx, result, method, args...)
	return err
}

func (c *rpcClient) BatchCallContext(ctx context.Context, b []rpc.BatchElem) error {
	err := c.rpc.BatchCallContext(ctx, b)
	return err
}

func toBlockNumArg(number *big.Int) string {
	if number == nil {
		return "latest"
	}
	if number.Sign() >= 0 {
		return hexutil.EncodeBig(number)
	}
	return rpc.BlockNumber(number.Int64()).String()
}

func toFilterArg(q ethereum.FilterQuery) (interface{}, error) {
	arg := map[string]interface{}{"address": q.Addresses, "topics": q.Topics}
	if q.BlockHash != nil {
		arg["blockHash"] = *q.BlockHash
		if q.FromBlock != nil || q.ToBlock != nil {
			return nil, errors.New("cannot specify both BlockHash and FromBlock/ToBlock")
		}
	} else {
		if q.FromBlock == nil {
			arg["fromBlock"] = "0x0"
		} else {
			arg["fromBlock"] = toBlockNumArg(q.FromBlock)
		}
		arg["toBlock"] = toBlockNumArg(q.ToBlock)
	}
	return arg, nil
}

func IsURLAvailable(address string) bool {
	u, err := url.Parse(address)
	if err != nil {
		return false
	}
	addr := u.Host
	if u.Port() == "" {
		switch u.Scheme {
		case "http", "ws":
			addr += ":80"
		case "https", "wss":
			addr += ":443"
		default:
			return true
		}
	}
	conn, err := net.DialTimeout("tcp", addr, 5*time.Second)
	if err != nil {
		return false
	}
	conn.Close()
	return true
}

func DialEthClientWithTimeout(ctx context.Context, url string, disableHTTP2 bool) (
	*ethclient.Client, error) {
	ctxt, cancel := context.WithTimeout(ctx, defaultDialTimeout)
	defer cancel()
	if strings.HasPrefix(url, "http") {
		httpClient := new(http.Client)
		if disableHTTP2 {
			log.Debug("Disabled HTTP/2 support in  eth client")
			httpClient.Transport = &http.Transport{
				TLSNextProto: make(map[string]func(authority string, c *tls.Conn) http.RoundTripper),
			}
		}
		rpcClient, err := rpc.DialHTTPWithClient(url, httpClient)
		if err != nil {
			return nil, err
		}
		return ethclient.NewClient(rpcClient), nil
	}
	return ethclient.DialContext(ctxt, url)
}

func toCallArg(msg ethereum.CallMsg) interface{} {
	arg := map[string]interface{}{
		"from": msg.From,
		"to":   msg.To,
	}
	if len(msg.Data) > 0 {
		arg["input"] = hexutil.Bytes(msg.Data)
	}
	if msg.Value != nil {
		arg["value"] = (*hexutil.Big)(msg.Value)
	}
	if msg.Gas != 0 {
		arg["gas"] = hexutil.Uint64(msg.Gas)
	}
	if msg.GasPrice != nil {
		arg["gasPrice"] = (*hexutil.Big)(msg.GasPrice)
	}
	return arg
}
