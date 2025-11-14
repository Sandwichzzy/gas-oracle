package worker

import (
	"context"
	"fmt"
	"time"

	"github.com/ethereum/go-ethereum/log"
	gresty "github.com/go-resty/resty/v2"
	"github.com/pkg/errors"

	"github.com/Sandwichzzy/gas-oracle/common/tasks"
	"github.com/Sandwichzzy/gas-oracle/database"
)

var errMarketHTTPError = errors.New("Skyeye market price  http error")

type Symbols struct {
	Name    string
	Decimal uint8
}

type WorkerHandleConfig struct {
	BaseUrl      string
	LoopInterval time.Duration
	SymbolList   []Symbols
}

type WorkerHandle struct {
	db             *database.DB
	wConf          *WorkerHandleConfig
	client         *gresty.Client // HTTP客户端
	resourceCtx    context.Context
	resourceCancel context.CancelFunc
	tasks          tasks.Group
}

func NewWorkerHandle(db *database.DB, wConf *WorkerHandleConfig, shutdown context.CancelCauseFunc) (*WorkerHandle, error) {
	client := gresty.New()
	client.SetBaseURL(wConf.BaseUrl)
	client.OnAfterResponse(func(c *gresty.Client, r *gresty.Response) error {
		statusCode := r.StatusCode()
		if statusCode >= 400 {
			method := r.Request.Method
			url := r.Request.URL
			return fmt.Errorf("%d cannot %s %s: %w", statusCode, method, url, errMarketHTTPError)
		}
		return nil
	})
	resCtx, resCancel := context.WithCancel(context.Background())
	return &WorkerHandle{
		db:             db,
		wConf:          wConf,
		client:         client,
		resourceCtx:    resCtx,
		resourceCancel: resCancel,
		tasks: tasks.Group{
			HandleCrit: func(err error) {
				shutdown(fmt.Errorf("critical error in worker handle processor: %w", err))
			},
		},
	}, nil
}

func (sh *WorkerHandle) Close() error {
	sh.resourceCancel()
	return sh.tasks.Wait()
}

func (sh *WorkerHandle) Start() error {
	workerTicker := time.NewTicker(sh.wConf.LoopInterval)
	sh.tasks.Go(func() error {
		for range workerTicker.C {
			err := sh.onProcessMarkerPrice()
			if err != nil {
				log.Error("process market price fail", "err", err)
				return err
			}
		}
		return nil
	})
	return nil
}

func (sh *WorkerHandle) onProcessMarkerPrice() error {
	for _, symbol := range sh.wConf.SymbolList {
		var resultData ResultData
		response, err := sh.client.R().SetQueryParam("symbol", symbol.Name).SetResult(&resultData).Get("api/v1/ccxt/price")
		if err != nil {
			return fmt.Errorf("cannot get %s market price: %w", symbol.Name, err)
		}
		if response.StatusCode() != 200 {
			return errors.New("get market price fail")
		}
		log.Info("get token marker price success", "Ok", resultData.Ok, "code", resultData.Code)
		if resultData.Ok {
			returnPriceData := resultData.Result

			log.Info("token marker price success", "symbol", symbol.Name, "code", returnPriceData.Price)

			tokenPrice := &database.TokenPrice{
				TokenName:   returnPriceData.BaseAsset,
				TokenSymbol: symbol.Name,
				Decimal:     symbol.Decimal,
				MarketPrice: fmt.Sprintf("%f", returnPriceData.Price),
				Timestamp:   uint64(time.Now().Unix()),
			}
			err := sh.db.TokenPrice.StoreOrUpdateTokenPrice(tokenPrice)
			if err != nil {
				log.Error("Store or update token price fail", "err", err)
				return err
			}
		}
	}
	return nil
}
