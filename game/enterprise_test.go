package game

import (
	"MiningGame/game/miners"
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func newTestEnterprise(t *testing.T, balance int) *Enterprise {
	t.Helper()
	e := NewEnterprise()
	t.Cleanup(e.Stop)
	e.mutex.Lock()
	e.balance = balance
	e.mutex.Unlock()
	return e
}

func waitFor(t *testing.T, check func() bool) {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if check() {
			return
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatal("condition did not become true")
}

func TestHireRejectsUnknownClassAndInsufficientBalance(t *testing.T) {
	e := newTestEnterprise(t, 4)
	if _, err := e.HireNewMiner("unknown"); !errors.Is(err, miners.NoSuchMinerType) {
		t.Fatalf("unknown miner: %v", err)
	}
	if _, err := e.HireNewMiner(miners.MinerTypeSmall); !errors.Is(err, miners.NotEnoughMoneyForHiringError) {
		t.Fatalf("insufficient balance: %v", err)
	}
	info := e.Info()
	if info.Balance != 4 || len(info.Miners) != 0 || info.Hired[miners.MinerTypeSmall] != 0 {
		t.Fatalf("failed hires changed enterprise: %+v", info)
	}
}

func TestConcurrentHiringCannotOverspend(t *testing.T) {
	e := newTestEnterprise(t, 450)
	var hired atomic.Int32
	var workers sync.WaitGroup
	for range 32 {
		workers.Go(func() {
			_, err := e.HireNewMiner(miners.MinerTypeStrong)
			if err == nil {
				hired.Add(1)
			} else if !errors.Is(err, miners.NotEnoughMoneyForHiringError) {
				t.Errorf("unexpected hire error: %v", err)
			}
		})
	}
	workers.Wait()
	info := e.Info()
	if hired.Load() != 1 || info.Balance != 0 || len(info.Miners) != 1 || info.Hired[miners.MinerTypeStrong] != 1 {
		t.Fatalf("concurrent hires: succeeded=%d, state=%+v", hired.Load(), info)
	}
}

func TestHiringAndBuyingShareOneBalance(t *testing.T) {
	e := newTestEnterprise(t, 3000)
	var hired, purchased atomic.Int32
	var workers sync.WaitGroup
	for range 20 {
		workers.Go(func() {
			_, err := e.HireNewMiner(miners.MinerTypeSmall)
			if err == nil {
				hired.Add(1)
			} else if !errors.Is(err, miners.NotEnoughMoneyForHiringError) {
				t.Errorf("unexpected hire error: %v", err)
			}
		})
		workers.Go(func() {
			_, err := e.BuyEquipment(EquipmentPickaxes)
			if err == nil {
				purchased.Add(1)
			} else if !errors.Is(err, ErrNotEnoughCoal) && !errors.Is(err, ErrEquipmentAlreadyPurchased) {
				t.Errorf("unexpected purchase error: %v", err)
			}
		})
	}
	workers.Wait()
	remaining := 3000 - int(hired.Load())*5 - int(purchased.Load())*3000
	if remaining < 0 || purchased.Load() > 1 || e.Info().Balance != remaining {
		t.Fatalf("balance=%d, hires=%d, purchases=%d", e.Info().Balance, hired.Load(), purchased.Load())
	}
}

func TestMinerFilteringAndSnapshots(t *testing.T) {
	e := newTestEnterprise(t, 55)
	for _, minerType := range []miners.MinerType{miners.MinerTypeSmall, miners.MinerTypeNormal} {
		if _, err := e.HireNewMiner(minerType); err != nil {
			t.Fatal(err)
		}
	}
	filtered, err := e.Miners(miners.MinerTypeNormal)
	if err != nil || len(filtered) != 1 || filtered[0].Info.Type != miners.MinerTypeNormal {
		t.Fatalf("filter result=%+v, err=%v", filtered, err)
	}
	if _, err := e.Miners("unknown"); !errors.Is(err, miners.NoSuchMinerType) {
		t.Fatalf("unknown filter accepted: %v", err)
	}
	info := e.Info()
	info.Miners[0].Info.Energy = 0
	info.Hired[miners.MinerTypeSmall] = 999
	info.Equipment[0].Purchased = true
	unchanged := e.Info()
	if unchanged.Miners[0].Info.Energy != 30 || unchanged.Hired[miners.MinerTypeSmall] != 1 || unchanged.Equipment[0].Purchased {
		t.Fatalf("mutating a snapshot changed enterprise: %+v", unchanged)
	}
}

func TestPassiveIncomeAndStrongMinerProgression(t *testing.T) {
	e := newTestEnterprise(t, 450)
	state, err := e.HireNewMiner(miners.MinerTypeStrong)
	if err != nil {
		t.Fatal(err)
	}
	if state.Info.Energy != 60 || state.Info.Coal != 10 {
		t.Fatalf("initial state: %+v", state)
	}
	waitFor(t, func() bool {
		info := e.Info()
		return len(info.Miners) == 1 && info.Miners[0].Info.Energy == 58 && info.Miners[0].Info.Coal == 16 && info.Balance >= 25
	})
	e.Stop()
	info := e.Info()
	if info.Balance != 25 || len(info.Miners) != 0 || info.Hired[miners.MinerTypeStrong] != 1 {
		t.Fatalf("expected 10+13 mined and 2 passive coal: %+v", info)
	}
	if e.ctx.Err() == nil {
		t.Fatal("enterprise context was not cancelled")
	}
	if state.Info.Energy != 60 || state.Info.Coal != 10 {
		t.Fatalf("initial snapshot changed: %+v", state)
	}
}

type shortMiner struct {
	info miners.MinerInfo
}

func (m *shortMiner) Info() miners.MinerInfo {
	return m.info
}

func (m *shortMiner) Run(ctx context.Context) <-chan miners.Coal {
	channel := make(chan miners.Coal, m.info.Energy)
	for range m.info.Energy {
		channel <- miners.Coal{}
	}
	close(channel)
	return channel
}

func TestExhaustedMinerIsRemovedAndHiringHistoryRemains(t *testing.T) {
	e := newTestEnterprise(t, 0)
	miner := &shortMiner{info: miners.MinerInfo{Type: miners.MinerTypeSmall, Energy: 2}}
	e.mutex.Lock()
	e.miners[1] = miner
	e.minerStates[1] = miner.Info()
	e.hired[miners.MinerTypeSmall] = 1
	e.workers.Add(1)
	go e.watchForMiner(1, miner.Run(e.ctx))
	e.mutex.Unlock()
	waitFor(t, func() bool { return len(e.Info().Miners) == 0 })
	if e.Info().Hired[miners.MinerTypeSmall] != 1 {
		t.Fatal("exhausted miner disappeared from hiring history")
	}
}

func TestEquipmentValidationAndCatalog(t *testing.T) {
	e := newTestEnterprise(t, 3000)
	if _, err := e.BuyEquipment("unknown"); !errors.Is(err, ErrUnknownEquipment) {
		t.Fatalf("unknown equipment: %v", err)
	}
	if _, err := e.BuyEquipment(EquipmentCarts); !errors.Is(err, ErrNotEnoughCoal) {
		t.Fatalf("insufficient balance: %v", err)
	}
	purchase, err := e.BuyEquipment(EquipmentPickaxes)
	if err != nil || !purchase.Purchased || purchase.Cost != 3000 {
		t.Fatalf("purchase=%+v, err=%v", purchase, err)
	}
	if _, err := e.BuyEquipment(EquipmentPickaxes); !errors.Is(err, ErrEquipmentAlreadyPurchased) {
		t.Fatalf("repeated purchase: %v", err)
	}
	catalog := e.Equipment()
	if len(catalog) != 3 || catalog[0].Cost != 3000 || catalog[1].Cost != 15000 || catalog[2].Cost != 50000 || e.Info().Balance != 0 {
		t.Fatalf("catalog=%+v, balance=%d", catalog, e.Info().Balance)
	}
}

func TestCompleteRequiresAllEquipmentAndStopsWorkers(t *testing.T) {
	e := newTestEnterprise(t, 68450)
	if _, err := e.Complete(); !errors.Is(err, ErrEquipmentIncomplete) {
		t.Fatalf("incomplete game finished: %v", err)
	}
	if _, err := e.HireNewMiner(miners.MinerTypeStrong); err != nil {
		t.Fatal(err)
	}
	for _, equipment := range e.Equipment() {
		if _, err := e.BuyEquipment(equipment.Type); err != nil {
			t.Fatal(err)
		}
	}
	goalInfo := e.Info()
	if goalInfo.GoalReachedAt == nil || goalInfo.GoalDuration < 0 {
		t.Fatalf("last purchase did not record goal time: %+v", goalInfo)
	}
	waitFor(t, func() bool { return e.Info().Duration > goalInfo.GoalDuration })
	started := time.Now()
	info, err := e.Complete()
	if err != nil {
		t.Fatal(err)
	}
	if time.Since(started) > 500*time.Millisecond {
		t.Fatal("completion waited for next mining tick")
	}
	if !info.Completed || info.FinishedAt == nil || len(info.Miners) != 0 || info.Balance != 0 || info.Hired[miners.MinerTypeStrong] != 1 {
		t.Fatalf("final state: %+v", info)
	}
	if info.Duration != info.FinishedAt.Sub(info.StartedAt) {
		t.Fatalf("duration=%v does not match timestamps", info.Duration)
	}
	if info.GoalDuration != goalInfo.GoalDuration || !info.GoalReachedAt.Equal(*goalInfo.GoalReachedAt) || info.Duration <= info.GoalDuration {
		t.Fatalf("goal and completion times: goal=%+v, completed=%+v", goalInfo, info)
	}
	if _, err := e.HireNewMiner(miners.MinerTypeSmall); !errors.Is(err, ErrGameStopped) {
		t.Fatalf("hiring after completion: %v", err)
	}
	if _, err := e.BuyEquipment(EquipmentPickaxes); !errors.Is(err, ErrGameStopped) {
		t.Fatalf("buying after completion: %v", err)
	}
	if _, err := e.Complete(); !errors.Is(err, ErrGameStopped) {
		t.Fatalf("repeated completion: %v", err)
	}
	time.Sleep(time.Millisecond)
	if next := e.Info(); next.Duration != info.Duration || next.Balance != info.Balance {
		t.Fatalf("completed game changed: %+v", next)
	}
	*info.FinishedAt = time.Time{}
	*info.GoalReachedAt = time.Time{}
	if e.Info().FinishedAt.IsZero() {
		t.Fatal("completion timestamp is shared with caller")
	}
	if e.Info().GoalReachedAt.IsZero() {
		t.Fatal("goal timestamp is shared with caller")
	}
	e.Stop()
}

func TestConcurrentStopAndHire(t *testing.T) {
	e := newTestEnterprise(t, 1000)
	var workers sync.WaitGroup
	for range 20 {
		workers.Go(func() {
			_, err := e.HireNewMiner(miners.MinerTypeSmall)
			if err != nil && !errors.Is(err, ErrGameStopped) {
				t.Errorf("unexpected hire error: %v", err)
			}
		})
		workers.Go(e.Stop)
	}
	workers.Wait()
	if len(e.Info().Miners) != 0 {
		t.Fatal("workers remain after Stop")
	}
}

type delayedMiner struct {
	cancelled chan struct{}
	release   chan struct{}
}

func (m *delayedMiner) Info() miners.MinerInfo {
	return miners.MinerInfo{Type: miners.MinerTypeSmall, Energy: 1}
}

func (m *delayedMiner) Run(ctx context.Context) <-chan miners.Coal {
	channel := make(chan miners.Coal)
	go func() {
		<-ctx.Done()
		close(m.cancelled)
		<-m.release
		close(channel)
	}()
	return channel
}

func TestStopWaitsForMinerChannelToClose(t *testing.T) {
	e := newTestEnterprise(t, 0)
	miner := &delayedMiner{cancelled: make(chan struct{}), release: make(chan struct{})}
	var release sync.Once
	defer release.Do(func() { close(miner.release) })
	e.mutex.Lock()
	e.miners[1] = miner
	e.minerStates[1] = miner.Info()
	e.workers.Add(1)
	go e.watchForMiner(1, miner.Run(e.ctx))
	e.mutex.Unlock()
	stopped := make(chan struct{})
	go func() {
		e.Stop()
		close(stopped)
	}()
	select {
	case <-miner.cancelled:
	case <-time.After(time.Second):
		t.Fatal("Stop did not cancel miner context")
	}
	select {
	case <-stopped:
		t.Fatal("Stop returned while miner channel remained open")
	case <-time.After(20 * time.Millisecond):
	}
	release.Do(func() { close(miner.release) })
	select {
	case <-stopped:
	case <-time.After(time.Second):
		t.Fatal("Stop did not return after miner channel closed")
	}
}
