package http

import (
	"MiningGame/game"
	"MiningGame/game/miners"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"sync"
)

type GameHandler struct {
	mu         sync.RWMutex
	enterprise *game.Enterprise
	closed     bool
	done       chan struct{}
	doneOnce   sync.Once
}

func NewGameHandler() *GameHandler {
	return &GameHandler{done: make(chan struct{})}
}

func (h *GameHandler) Done() <-chan struct{} {
	return h.done
}

func (h *GameHandler) Close() {
	h.mu.Lock()
	h.closed = true
	enterprise := h.enterprise
	h.mu.Unlock()
	if enterprise != nil {
		enterprise.Stop()
	}
}

func (h *GameHandler) getEnterprise(w http.ResponseWriter) *game.Enterprise {
	h.mu.RLock()
	enterprise, closed := h.enterprise, h.closed
	h.mu.RUnlock()
	if closed {
		writeError(w, http.StatusConflict, "Игра завершена")
		return nil
	}
	if enterprise == nil {
		writeError(w, http.StatusConflict, "Сначала запустите предприятие")
	}
	return enterprise
}

func decodeRequest[T any](w http.ResponseWriter, r *http.Request) (T, bool) {
	var zero T
	var request *T
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&request); err != nil {
		writeError(w, http.StatusBadRequest, "Некорректный JSON: "+err.Error())
		return zero, false
	}
	if request == nil {
		writeError(w, http.StatusBadRequest, "Тело запроса должно содержать один JSON-объект")
		return zero, false
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		writeError(w, http.StatusBadRequest, "Тело запроса должно содержать один JSON-объект")
		return zero, false
	}
	return *request, true
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(value); err != nil {
		log.Printf("Не удалось отправить JSON-ответ: %v", err)
	}
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, struct {
		Error string `json:"error"`
	}{Error: message})
}

func writeGameError(w http.ResponseWriter, err error) {
	status := http.StatusInternalServerError
	switch {
	case errors.Is(err, miners.NoSuchMinerType), errors.Is(err, game.ErrUnknownEquipment):
		status = http.StatusBadRequest
	case errors.Is(err, miners.NotEnoughMoneyForHiringError),
		errors.Is(err, game.ErrNotEnoughCoal),
		errors.Is(err, game.ErrEquipmentAlreadyPurchased),
		errors.Is(err, game.ErrEquipmentIncomplete),
		errors.Is(err, game.ErrGameStopped):
		status = http.StatusConflict
	}
	writeError(w, status, err.Error())
}
