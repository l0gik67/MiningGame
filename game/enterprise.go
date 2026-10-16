package game

import (
	"MiningGame/game/miners"
	"context"
	"sync"
	"sync/atomic"
	"time"
)

type Enterprise struct {
	miners      map[int]miners.Miner
	amount      atomic.Int64
	ctx         context.Context
	cancelFunc context.CancelFunc
	nextMinerId int
	mutex       sync.RWMutex
}

func NewEnterprise() *Enterprise{
	ctx, cancelFunc := context.WithCancel(context.Background())
	e := &Enterprise{
		miners : make(map[int]miners.Miner),
		amount : atomic.Int64{},
		ctx : ctx,
		cancelFunc : cancelFunc,
		nextMinerId : 1,
		mutex : sync.RWMutex{},
	}

	go func() {
		ticker := time.NewTicker(time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				e.amount.Add(1)
			}
		}
	} ()

	return e
}

func (e *Enterprise) HireNewMiner(minerType miners.MinerType) (miners.Miner, error) {
	var miner miners.Miner

	e.mutex.Lock()
	if err := miners.CheckAvailabilityOfPayment(minerType, int(e.amount.Load())); err != nil {
		e.mutex.Unlock()
		return nil, err
	}
	e.amount.Add(int64(-miners.GetMinerCost(minerType)))
	e.mutex.Unlock()

	switch minerType {
	case miners.MinerTypeSmall:
		miner = miners.NewSmallMiner()
	case miners.MinerTypeNormal:
		miner = miners.NewNormalMiner()
	case miners.MinerTypeStrong:
		miner = miners.NewStrongMiner()
	}

	e.mutex.Lock()

	e.miners[e.nextMinerId] = miner
	channel := miner.Run(e.ctx)
	go e.watchForMiner(e.nextMinerId, channel)
	e.nextMinerId++

	e.mutex.Unlock()

	return miner, nil
}

func (e *Enterprise) watchForMiner(minerId int, channel <-chan miners.Coal) {
	defer func() {
		e.mutex.Lock()
		delete(e.miners, minerId)
		e.mutex.Unlock()
	}()
	for {
		select {
		case <-e.ctx.Done():
			return
		case coal, ok := <-channel:
			if !ok {
				return
			}
			e.amount.Add(int64(coal.GetCount()))
		}
	}
}
