package miners

import (
	"context"
	"time"
)

type smallMiner struct {
	info MinerConfig
}

func NewSmallMiner() Miner {
	return &smallMiner{
		info: minerConfigs[MinerTypeSmall],
	}
}

func (m *smallMiner) Run(ctx context.Context) <-chan Coal {
	channel := make(chan Coal, m.info.Energy)
	go func(ctx context.Context) {
		ticker := time.NewTicker(m.info.Interval)
		defer ticker.Stop()
		defer close(channel)
		for m.info.Energy > 0 {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				m.info.Energy--
				channel <- Coal{count: m.info.Coal}
			}
		}
	}(ctx)

	return channel
}

func (m *smallMiner) Info() MinerConfig {
	return m.info
}
