package http

import (
	"MiningGame/game/miners"
	"MiningGame/http/dto"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"testing/synctest"
	"time"
)

func call(t *testing.T, router http.Handler, method, path, body string, want int) *httptest.ResponseRecorder {
	t.Helper()
	w := httptest.NewRecorder()
	router.ServeHTTP(w, httptest.NewRequest(method, path, strings.NewReader(body)))
	if w.Code != want {
		t.Fatalf("%s %s: status = %d, want %d; body = %s", method, path, w.Code, want, w.Body.String())
	}
	if !strings.HasPrefix(w.Header().Get("Content-Type"), "application/json") {
		t.Fatalf("%s %s: response is not JSON", method, path)
	}
	if want >= 400 {
		var response struct {
			Error string `json:"error"`
		}
		if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil || response.Error == "" {
			t.Fatalf("invalid error response: %s", w.Body.String())
		}
	}
	return w
}

func responseAs[T any](t *testing.T, response *httptest.ResponseRecorder) T {
	t.Helper()
	var value T
	if err := json.Unmarshal(response.Body.Bytes(), &value); err != nil {
		t.Fatal(err)
	}
	return value
}

func TestCatalogAndStartRequired(t *testing.T) {
	h := NewGameHandler()
	defer h.Close()
	router := h.Router()
	response := call(t, router, http.MethodGet, "/miners/types", "", http.StatusOK)
	catalog := responseAs[[]dto.MinerDto](t, response)
	want := miners.GetMinerTypes()
	if len(catalog) != 3 {
		t.Fatalf("catalog has %d types", len(catalog))
	}
	for i, config := range want {
		if catalog[i] != dto.MinerDto(config) {
			t.Fatalf("catalog entry = %+v, want %+v", catalog[i], config)
		}
	}
	for _, request := range []struct{ method, path, body string }{
		{http.MethodGet, "/miners", ""},
		{http.MethodPost, "/miners", `{"miner_type":"small_miner"}`},
		{http.MethodGet, "/equipment", ""},
		{http.MethodPost, "/equipment", `{"type":"pickaxes"}`},
		{http.MethodGet, "/enterprise", ""},
		{http.MethodPost, "/enterprise/completion", ""},
	} {
		call(t, router, request.method, request.path, request.body, http.StatusConflict)
	}
	for _, method := range []string{http.MethodPost, http.MethodPut, http.MethodDelete} {
		call(t, router, method, "/miners/types", `{}`, http.StatusMethodNotAllowed)
	}
	call(t, router, http.MethodGet, "/missing", "", http.StatusNotFound)
	call(t, router, http.MethodPost, "/enterprise/beginning", "", http.StatusCreated)
	call(t, router, http.MethodPost, "/enterprise/beginning", "", http.StatusConflict)
	call(t, router, http.MethodGet, "/miners", "", http.StatusOK)
	call(t, router, http.MethodPost, "/enterprise/completion", "", http.StatusConflict)
	call(t, router, http.MethodPost, "/enterprise/complition", "", http.StatusConflict)
	select {
	case <-h.Done():
		t.Fatal("failed completion closed Done")
	default:
	}
}

func TestRequestValidation(t *testing.T) {
	h := NewGameHandler()
	defer h.Close()
	router := h.Router()
	call(t, router, http.MethodPost, "/enterprise/beginning", "", http.StatusCreated)
	for _, body := range []string{
		"", "null", "[]", `"small_miner"`, "{", "{}", `{"miner_type":1}`,
		`{"miner_type":"unknown"}`, `{"miner_type":"small_miner","energy":100}`,
		`{"miner_type":"small_miner"} {}`, `{"miner_type":"small_miner"} garbage`,
		`{"miner_type":"small_miner","cost":0}`,
	} {
		t.Run("miner_"+body, func(t *testing.T) {
			call(t, router, http.MethodPost, "/miners", body, http.StatusBadRequest)
		})
	}
	for _, body := range []string{
		"", "null", "[]", "{}", `{"type":"unknown"}`, `{"type":false}`,
		`{"type":"pickaxes","cost":0}`, `{"type":"pickaxes"} null`,
	} {
		t.Run("equipment_"+body, func(t *testing.T) {
			call(t, router, http.MethodPost, "/equipment", body, http.StatusBadRequest)
		})
	}
	for _, path := range []string{
		"/miners?type=unknown", "/miners?type=", "/miners?type=small_miner&type=strong_miner",
	} {
		call(t, router, http.MethodGet, path, "", http.StatusBadRequest)
	}
	call(t, router, http.MethodPost, "/miners", `{"miner_type":"small_miner"}`, http.StatusConflict)
	call(t, router, http.MethodPost, "/equipment", `{"type":"pickaxes"}`, http.StatusConflict)
}

func TestGameLifecycle(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		h := NewGameHandler()
		defer h.Close()
		router := h.Router()
		initial := responseAs[dto.EnterpriseDto](t, call(t, router, http.MethodPost, "/enterprise/beginning", "", http.StatusCreated))
		if initial.Balance != 0 || len(initial.Miners) != 0 || len(initial.Equipment) != 3 || initial.Completed {
			t.Fatalf("invalid initial state: %+v", initial)
		}
		synctest.Wait()
		time.Sleep(5 * time.Second)
		synctest.Wait()
		miner := responseAs[dto.MinerStateDto](t, call(t, router, http.MethodPost, "/miners", `{"miner_type":"small_miner"}`, http.StatusCreated))
		if miner.ID == 0 || miner.Type != miners.MinerTypeSmall || miner.Energy != 30 || miner.Cost != 5 {
			t.Fatalf("invalid hired miner: %+v", miner)
		}
		workers := responseAs[[]dto.MinerStateDto](t, call(t, router, http.MethodGet, "/miners?type=small_miner", "", http.StatusOK))
		if len(workers) != 1 || workers[0].ID != miner.ID {
			t.Fatalf("small miner filter = %+v", workers)
		}
		other := responseAs[[]dto.MinerStateDto](t, call(t, router, http.MethodGet, "/miners?type=normal_miner", "", http.StatusOK))
		if other == nil || len(other) != 0 {
			t.Fatalf("normal miner filter = %+v", other)
		}
		synctest.Wait()
		time.Sleep(90 * time.Second)
		synctest.Wait()
		state := responseAs[dto.EnterpriseDto](t, call(t, router, http.MethodGet, "/enterprise", "", http.StatusOK))
		if len(state.Miners) != 0 || state.Hired[miners.MinerTypeSmall] != 1 || state.Balance != 120 {
			t.Fatalf("invalid state after miner retirement: %+v", state)
		}
		time.Sleep(67880 * time.Second)
		synctest.Wait()
		for _, equipment := range []string{"pickaxes", "ventilation", "carts"} {
			item := responseAs[dto.EquipmentDto](t, call(t, router, http.MethodPost, "/equipment", `{"type":"`+equipment+`"}`, http.StatusOK))
			if string(item.Type) != equipment || !item.Purchased {
				t.Fatalf("invalid purchased equipment: %+v", item)
			}
		}
		call(t, router, http.MethodPost, "/equipment", `{"type":"pickaxes"}`, http.StatusConflict)
		equipment := responseAs[[]dto.EquipmentDto](t, call(t, router, http.MethodGet, "/equipment", "", http.StatusOK))
		for _, item := range equipment {
			if !item.Purchased {
				t.Fatalf("equipment not purchased: %+v", item)
			}
		}
		time.Sleep(10 * time.Second)
		synctest.Wait()
		call(t, router, http.MethodPost, "/miners", `{"miner_type":"small_miner"}`, http.StatusCreated)
		synctest.Wait()
		final := responseAs[dto.EnterpriseDto](t, call(t, router, http.MethodPost, "/enterprise/completion", "", http.StatusOK))
		if !final.Completed || final.FinishedAt == nil || final.GoalReachedAt == nil || final.Balance != 5 || len(final.Miners) != 0 || final.Hired[miners.MinerTypeSmall] != 2 || final.Duration-final.GoalDuration != 10*time.Second {
			t.Fatalf("invalid final state: %+v", final)
		}
		select {
		case <-h.Done():
		default:
			t.Fatal("successful completion did not close Done")
		}
		time.Sleep(10 * time.Second)
		synctest.Wait()
		stopped := responseAs[dto.EnterpriseDto](t, call(t, router, http.MethodGet, "/enterprise", "", http.StatusOK))
		if stopped.Balance != final.Balance || stopped.Duration != final.Duration {
			t.Fatalf("state changed after completion: %+v", stopped)
		}
		call(t, router, http.MethodPost, "/miners", `{"miner_type":"small_miner"}`, http.StatusConflict)
	})
}

func TestConcurrentStartAndClose(t *testing.T) {
	h := NewGameHandler()
	defer h.Close()
	router := h.Router()
	var wg sync.WaitGroup
	statuses := make(chan int, 16)
	for range 16 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			w := httptest.NewRecorder()
			router.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/enterprise/beginning", nil))
			statuses <- w.Code
		}()
	}
	wg.Wait()
	close(statuses)
	started := 0
	for status := range statuses {
		if status == http.StatusCreated {
			started++
		} else if status != http.StatusConflict {
			t.Fatalf("unexpected start status: %d", status)
		}
	}
	if started != 1 {
		t.Fatalf("started %d enterprises", started)
	}
	h.Close()
	h.Close()
	call(t, router, http.MethodPost, "/enterprise/beginning", "", http.StatusConflict)
	call(t, router, http.MethodGet, "/enterprise", "", http.StatusConflict)
	closed := NewGameHandler()
	closed.Close()
	call(t, closed.Router(), http.MethodPost, "/enterprise/beginning", "", http.StatusConflict)
}
