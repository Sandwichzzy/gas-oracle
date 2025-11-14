package grpc

import (
	"context"
	"fmt"
	"math/big"
	"strconv"

	"github.com/ethereum/go-ethereum/log"
	"github.com/pkg/errors"

	"github.com/Sandwichzzy/gas-oracle/services/grpc/gasFeePb"
)

func (ms *TokenPriceRpcService) GetTokenPriceAndGasByChainId(ctx context.Context, in *gasFeePb.TokenGasPriceRequest) (*gasFeePb.TokenGasPriceResponse, error) {
	gasFee, err := ms.db.GasFee.QueryGasFees(strconv.FormatUint(in.ChainId, 10))
	if err != nil {
		log.Error("Query gas fee fail", "err", err)
		return nil, err
	}

	nativeTokenPrice, err := ms.db.TokenPrice.QueryTokenPrices(gasFee.TokenName)
	if err != nil {
		log.Error("Query native token price fail", "err", err)
		return nil, err
	}

	tokenPrice, err := ms.db.TokenPrice.QueryTokenPrices(in.Symbol)
	if err != nil {
		log.Error("Query token price fail", "err", err)
		return nil, err
	}

	log.Info("get gas fee success", "predictFee", gasFee.PredictFee, "tokenName", gasFee.TokenName, "decimal", gasFee.Decimal)
	log.Info("get token price success", "marketPrice", tokenPrice.MarketPrice)

	fGasFee, err := strconv.ParseFloat(gasFee.PredictFee, 64)
	if err != nil {
		log.Error("fee convert fail", "err", err)
		return nil, errors.New("fee convert fail")
	}

	log.Info("parse float success", "fGasFee", fGasFee)

	resultValue := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(gasFee.Decimal)), nil)
	value, _ := resultValue.Float64()

	nativeTokenMarketPrice, _ := strconv.ParseFloat(nativeTokenPrice.MarketPrice, 64)
	symbolMarketPrice, _ := strconv.ParseFloat(tokenPrice.MarketPrice, 64)

	//pFee = (基础Gas费用 / 10^小数位数) × (原生代币价格 / 目标代币价格)  目标代币支付 Gas 费用的预估金额：
	pFee := (fGasFee / value) * (nativeTokenMarketPrice / symbolMarketPrice)

	return &gasFeePb.TokenGasPriceResponse{
		ReturnCode:  100,
		Message:     "get gas fee success",
		PredictFee:  fmt.Sprintf("%.8f", pFee),
		Symbol:      in.Symbol,
		MarketPrice: tokenPrice.MarketPrice,
	}, nil
}
