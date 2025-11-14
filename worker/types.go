package worker

import "time"

type ResultData struct {
	Ok     bool    `json:"ok"`
	Code   int     `json:"code"`
	Result Message `json:"result"`
}

type Message struct {
	BaseAsset      string    `json:"base_asset"`
	Exchange       string    `json:"exchange"`
	Price          float64   `json:"price"`
	PriceChange24H float64   `json:"price_change_24h"`
	Volume24H      float64   `json:"volume_24h"`
	Timestamp      time.Time `json:"timestamp"`
}
