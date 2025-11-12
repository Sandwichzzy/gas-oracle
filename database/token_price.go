package database

import (
	"github.com/ethereum/go-ethereum/log"
	"github.com/google/uuid"
	"github.com/pkg/errors"
	"gorm.io/gorm"
)

type TokenPrice struct {
	GUID        uuid.UUID `json:"guid" gorm:"primaryKey;DEFAULT replace(uuid_generate_v4()::text,'-','');serializer:uuid"`
	TokenName   string    `json:"token_name"`
	TokenSymbol string    `json:"token_symbol"`
	Decimal     uint8     `json:"decimal"`
	MarketPrice string    `json:"market_price"`
	Timestamp   uint64    `json:"timestamp"`
}

func (TokenPrice) TableName() string {
	return "token_price"
}

type tokenPriceDB struct {
	gorm *gorm.DB
}

type TokenPriceDB interface {
	TokenPriceView
	StoreOrUpdateTokenPrice(msgHash *TokenPrice) error
}

type TokenPriceView interface {
	QueryTokenPrices(symbol string) (*TokenPrice, error)
}

func NewTokenPriceDB(db *gorm.DB) TokenPriceDB {
	return &tokenPriceDB{gorm: db}
}

func (db *tokenPriceDB) StoreOrUpdateTokenPrice(tokenPrice *TokenPrice) error {
	var tokenPriceRecord TokenPrice
	err := db.gorm.Table("token_price").Where("token_symbol = ?", tokenPrice.TokenSymbol).Take(&tokenPriceRecord).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			result := db.gorm.Table("token_price").Omit("guid").Create(tokenPrice)
			return result.Error
		}
	}
	tokenPriceRecord.MarketPrice = tokenPrice.MarketPrice
	tokenPriceRecord.Timestamp = tokenPrice.Timestamp
	err = db.gorm.Table("token_price").Save(tokenPriceRecord).Error
	if err != nil {
		log.Error("save token price record fail", "err", err)
		return err
	}
	return nil
}

func (db *tokenPriceDB) QueryTokenPrices(symbol string) (*TokenPrice, error) {
	var tokenPrice TokenPrice
	err := db.gorm.Table("token_price").Where("token_symbol = ?", symbol).Take(&tokenPrice).Error
	if err != nil {
		log.Error("get token price fail", "err", err)
		return nil, err
	}
	return &tokenPrice, nil
}
