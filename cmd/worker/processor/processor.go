package processor

import (
	"encoding/json"

	"choopong.com/n-rabbit-consumers/pkg/model"
	"choopong.com/n-rabbit-consumers/pkg/rabbitmq"
	logger "github.com/sirupsen/logrus"
)

const SUGGESTED_SHOP_INDEX_PREFIX = "suggested_shop_"
const SUGGESTED_SHOP_ALIAS = "suggested_shop"

const UPDATED_RK = "updated"
const FINISHED_RK = "finished"

type FinishedPayload struct {
	Timestamp string `json:"timestamp" validate:"required"`
}

type UpdatedPayload struct {
	SuggestedShops []SuggestedShop `json:"suggestedShops" validate:"required"`
	FinishedPayload
}

type SuggestedShop struct {
	ShopID     int     `json:"shopId" validate:"required"`
	ShopName   string  `json:"shopName" validate:"required"`
	PremiumId  *string `json:"premiumId"`
	SearchRate int     `json:"searchRate" validate:"required"`
}

type processor struct {
}

func NewProcessor() rabbitmq.Processor {
	return &processor{}
}

func (p *processor) Process(routingKey string, b []byte) (err error) {
	logger.Info("start process")
	var data model.Data
	err = json.Unmarshal(b, &data)
	if err != nil {
		logger.Error(err)
	}
	logger.Infof("ID: %d", data.ID)
	defer logger.Info("finish process")
	return nil
}

func (p *processor) Fallback(bytes []byte, err error) {
}
