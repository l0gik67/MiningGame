package game

import (
	"MiningGame/game/miners"
	"context"
	"sort"
	"sync"
	"time"
)

type MinerState struct {
	ID   int
	Info miners.MinerInfo
}

type EnterpriseInfo struct {
	Balance       int
	Miners        []MinerState
	Hired         map[miners.MinerType]int
	Equipment     []EquipmentInfo
	StartedAt     time.Time
	GoalReachedAt *time.Time
	GoalDuration  time.Duration
	FinishedAt    *time.Time
	Duration      time.Duration
	Completed     bool
}

type profitEvent struct {
	minerID  int
	amount   int
	finished bool
}

type Enterprise struct {
	miners        map[int]miners.Miner
	minerStates   map[int]miners.MinerInfo
	hired         map[miners.MinerType]int
	equipment     map[EquipmentType]bool
	balance       int
	ctx           context.Context
	cancelFunc    context.CancelFunc
	profits       chan profitEvent
	nextMinerID   int
	startedAt     time.Time
	goalReachedAt *time.Time
	finishedAt    *time.Time
	completed     bool
	stopped       bool
	mutex         sync.RWMutex
	workers       sync.WaitGroup
}

func NewEnterprise() *Enterprise {
	ctx, cancelFunc := context.WithCancel(context.Background())
	e := &Enterprise{
		miners:      make(map[int]miners.Miner),
		minerStates: make(map[int]miners.MinerInfo),
		hired: map[miners.MinerType]int{
			miners.MinerTypeSmall:  0,
			miners.MinerTypeNormal: 0,
			miners.MinerTypeStrong: 0,
		},
		equipment:   make(map[EquipmentType]bool),
		ctx:         ctx,
		cancelFunc:  cancelFunc,
		profits:     make(chan profitEvent),
		nextMinerID: 1,
		startedAt:   time.Now(),
	}
	e.workers.Add(2)
	go e.collectProfits()
	go e.passiveIncome()
	return e
}

func (e *Enterprise) HireNewMiner(minerType miners.MinerType) (MinerState, error) {
	e.mutex.Lock()
	defer e.mutex.Unlock()
	if e.stopped {
		return MinerState{}, ErrGameStopped
	}
	if err := miners.CheckAvailabilityOfPayment(minerType, e.balance); err != nil {
		return MinerState{}, err
	}
	var miner miners.Miner
	switch minerType {
	case miners.MinerTypeSmall:
		miner = miners.NewSmallMiner()
	case miners.MinerTypeNormal:
		miner = miners.NewNormalMiner()
	case miners.MinerTypeStrong:
		miner = miners.NewStrongMiner()
	}
	info := miner.Info()
	state := MinerState{ID: e.nextMinerID, Info: info}
	e.balance -= info.Cost
	e.miners[state.ID] = miner
	e.minerStates[state.ID] = info
	e.hired[minerType]++
	e.nextMinerID++
	e.workers.Add(1)
	go e.watchForMiner(state.ID, miner.Run(e.ctx))
	return state, nil
}

func (e *Enterprise) Miners(filter miners.MinerType) ([]MinerState, error) {
	if filter != "" {
		if err := miners.CheckAvailabilityOfPayment(filter, miners.GetMinerCost(filter)); err != nil {
			return nil, err
		}
	}
	e.mutex.RLock()
	defer e.mutex.RUnlock()
	return e.minersLocked(filter), nil
}

func (e *Enterprise) minersLocked(filter miners.MinerType) []MinerState {
	states := make([]MinerState, 0, len(e.minerStates))
	for id, info := range e.minerStates {
		if filter == "" || info.Type == filter {
			states = append(states, MinerState{ID: id, Info: info})
		}
	}
	sort.Slice(states, func(i, j int) bool { return states[i].ID < states[j].ID })
	return states
}

func (e *Enterprise) passiveIncome() {
	defer e.workers.Done()
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-e.ctx.Done():
			return
		case <-ticker.C:
			select {
			case e.profits <- profitEvent{amount: 1}:
			case <-e.ctx.Done():
				return
			}
		}
	}
}

func (e *Enterprise) watchForMiner(minerID int, channel <-chan miners.Coal) {
	defer e.workers.Done()
	for coal := range channel {
		select {
		case e.profits <- profitEvent{minerID: minerID, amount: coal.GetCount()}:
		case <-e.ctx.Done():
		}
	}
	select {
	case e.profits <- profitEvent{minerID: minerID, finished: true}:
	case <-e.ctx.Done():
	}
}

func (e *Enterprise) collectProfits() {
	defer e.workers.Done()
	for {
		select {
		case <-e.ctx.Done():
			return
		case event := <-e.profits:
			e.mutex.Lock()
			if !e.stopped {
				e.balance += event.amount
				if info, ok := e.minerStates[event.minerID]; ok {
					info.Energy--
					info.Coal += info.Progression
					if event.finished || info.Energy <= 0 {
						delete(e.miners, event.minerID)
						delete(e.minerStates, event.minerID)
					} else {
						e.minerStates[event.minerID] = info
					}
				}
			}
			e.mutex.Unlock()
		}
	}
}

func (e *Enterprise) Info() EnterpriseInfo {
	e.mutex.RLock()
	defer e.mutex.RUnlock()
	return e.infoLocked()
}

func (e *Enterprise) infoLocked() EnterpriseInfo {
	hired := make(map[miners.MinerType]int, len(e.hired))
	for minerType, count := range e.hired {
		hired[minerType] = count
	}
	duration := time.Since(e.startedAt)
	var finishedAt *time.Time
	var goalReachedAt *time.Time
	var goalDuration time.Duration
	if e.goalReachedAt != nil {
		goalReached := *e.goalReachedAt
		goalReachedAt = &goalReached
		goalDuration = goalReached.Sub(e.startedAt)
	}
	if e.finishedAt != nil {
		finished := *e.finishedAt
		finishedAt = &finished
		duration = finished.Sub(e.startedAt)
	}
	return EnterpriseInfo{
		Balance: e.balance, Miners: e.minersLocked(""), Hired: hired,
		Equipment: e.equipmentLocked(), StartedAt: e.startedAt,
		GoalReachedAt: goalReachedAt, GoalDuration: goalDuration,
		FinishedAt: finishedAt, Duration: duration, Completed: e.completed,
	}
}

func (e *Enterprise) Complete() (EnterpriseInfo, error) {
	e.mutex.Lock()
	if e.stopped {
		e.mutex.Unlock()
		return EnterpriseInfo{}, ErrGameStopped
	}
	for _, equipment := range equipmentCatalog {
		if !e.equipment[equipment.Type] {
			e.mutex.Unlock()
			return EnterpriseInfo{}, ErrEquipmentIncomplete
		}
	}
	e.completed = true
	e.stopLocked()
	e.mutex.Unlock()
	e.workers.Wait()
	return e.Info(), nil
}

func (e *Enterprise) Stop() {
	e.mutex.Lock()
	if !e.stopped {
		e.stopLocked()
	}
	e.mutex.Unlock()
	e.workers.Wait()
}

func (e *Enterprise) stopLocked() {
	e.stopped = true
	finished := time.Now()
	e.finishedAt = &finished
	e.cancelFunc()
	clear(e.miners)
	clear(e.minerStates)
}
