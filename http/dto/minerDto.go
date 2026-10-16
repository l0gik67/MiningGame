package dto

import (
	"MiningGame/game/miners"
	"time"
)

type MinerTypeDto struct {
	MinerType miners.MinerType `json:"miner_type"`
}

type MinerDto struct {
	Type        miners.MinerType `json:"type"`
	Cost        int              `json:"cost"`
	Energy      int              `json:"energy"`
	Coal        int              `json:"coal"`
	Interval    time.Duration    `json:"interval"`
	Progression int              `json:"progression"`
}
