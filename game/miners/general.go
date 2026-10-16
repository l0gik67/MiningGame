package miners

import (
	"context"
	"time"
)

type Coal struct {
	count int
}

func (c *Coal) GetCount() int {
	return c.count
}

type MinerType string

const (
	MinerTypeSmall  MinerType = "small_miner"
	MinerTypeNormal MinerType = "normal_miner"
	MinerTypeStrong MinerType = "strong_miner"
)

type MinerConfig struct {
	Type        MinerType
	Cost        int
	Energy      int
	Coal        int
	Interval    time.Duration
	Progression int
}

var minerConfigs = map[MinerType]MinerConfig{
	MinerTypeSmall: {
		Type:     MinerTypeSmall,
		Cost:     5,
		Energy:   30,
		Coal:     1,
		Interval: 3 * time.Second,
	},
	MinerTypeNormal: {
		Type:     MinerTypeNormal,
		Cost:     50,
		Energy:   45,
		Coal:     3,
		Interval: 2 * time.Second,
	},
	MinerTypeStrong: {
		Type:        MinerTypeStrong,
		Cost:        450,
		Energy:      60,
		Coal:        10,
		Interval:    time.Second,
		Progression: 3,
	},
}

type Miner interface {
	Run(ctx context.Context) <-chan Coal
	Info() MinerConfig
}

func CheckAvailabilityOfPayment(minerType MinerType, amount int) error {
	config, ok := minerConfigs[minerType]
	if !ok {
		return NoSuchMinerType
	}
	if amount >= config.Cost {
		return nil
	}
	return NotEnoughMoneyForHiringError
}

func GetMinerCost(minerType MinerType) int {
	return minerConfigs[minerType].Cost
}
