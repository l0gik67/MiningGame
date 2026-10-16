package miners

import (
	"context"
	"time"
)

type strongMiner struct {
	info MinerConfig
}

func NewStrongMiner() Miner {
	return &strongMiner{
		info: minerConfigs[MinerTypeStrong],
	}
}

func (miner *strongMiner) Run(ctx context.Context) <-chan Coal {
	channel := make(chan Coal, miner.info.Energy)

	go func() {
		ticker := time.NewTicker(miner.info.Interval)
		defer ticker.Stop()
		defer close(channel)

		for miner.info.Energy > 0 {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				channel <- Coal{count: miner.info.Coal}
				miner.info.Energy--
				miner.info.Coal += miner.info.Progression
			}
		}
	}()

	return channel
}

func (miner *strongMiner) Info() MinerConfig {
	return miner.info
}
