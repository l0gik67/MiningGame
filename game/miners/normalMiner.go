package miners

import (
	"context"
	"time"
)

type normalMiner struct {
	info MinerConfig
}

func NewNormalMiner() Miner {
	return &normalMiner{
		info: minerConfigs[MinerTypeNormal],
	}
}

func (miner *normalMiner) Run(ctx context.Context) <-chan Coal {
	channel := make(chan Coal, miner.info.Energy)

	go func(ctx context.Context) {
		ticker := time.NewTicker(miner.info.Interval)
		defer ticker.Stop()
		defer close(channel)
		for miner.info.Energy > 0 {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				channel <- Coal{miner.info.Coal}
				miner.info.Energy--
			}
		}
	}(ctx)

	return channel
}

func (miner *normalMiner) Info() MinerConfig {
	return miner.info
}
