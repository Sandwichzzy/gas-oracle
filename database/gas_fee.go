package database

import (
	"gorm.io/gorm"
	"math/big"

	"github.com/google/uuid"
	"github.com/pkg/errors"

	"github.com/ethereum/go-ethereum/log"
)

type GasFee struct {
	GUID       uuid.UUID `json:"guid" gorm:"primaryKey;DEFAULT replace(uuid_generate_v4()::text,'-','');serializer:uuid"`
	ChainId    *big.Int  `json:"chain_id" gorm:"serializer:u256"`
	TokenName  string    `json:"token_name"`
	Decimal    uint8     `json:"decimal"`
	PredictFee string    `json:"predict_fee"`
	Timestamp  uint64    `json:"timestamp"`
}

func (*GasFee) TableName() string {
	return "gas_fee"
}

type gasFeeDB struct {
	gorm *gorm.DB
}

type GasFeeDB interface {
	GasFeeView
	StoreOrUpdateGasFee(msgHash *GasFee) error
}

type GasFeeView interface {
	QueryGasFees(chainId string) (*GasFee, error)
}

func NewGasFeeDB(db *gorm.DB) GasFeeDB {
	return &gasFeeDB{gorm: db}
}

func (db *gasFeeDB) StoreOrUpdateGasFee(gasFee *GasFee) error {
	var gasFeeRecord GasFee
	err := db.gorm.Table("gas_fee").Where("chain_id = ?", gasFee.ChainId.String()).Take(&gasFeeRecord).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			result := db.gorm.Table("gas_fee").Omit("guid").Create(gasFee)
			return result.Error
		}
	}
	gasFeeRecord.TokenName = gasFee.TokenName
	gasFeeRecord.PredictFee = gasFee.PredictFee
	gasFeeRecord.Timestamp = gasFee.Timestamp
	gasFeeRecord.Decimal = gasFee.Decimal
	err = db.gorm.Table("gas_fee").Save(gasFeeRecord).Error
	if err != nil {
		log.Error("save gas fee record fail", "err", err)
		return err
	}
	return nil
}

func (db *gasFeeDB) QueryGasFees(chainId string) (*GasFee, error) {
	var gasFee GasFee
	err := db.gorm.Table("gas_fee").Where("chain_id = ?", chainId).Take(&gasFee).Error
	if err != nil {
		log.Error("get gas fee fail", "err", err)
		return nil, err
	}
	return &gasFee, nil
}
