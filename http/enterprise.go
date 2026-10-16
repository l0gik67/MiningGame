package http

import (
	"MiningGame/game"
	"MiningGame/http/dto"
	"net/http"
)

func (h *GameHandler) getEnterpriseInfo(w http.ResponseWriter, r *http.Request) {
	enterprise := h.getEnterprise(w)
	if enterprise == nil {
		return
	}
	writeJSON(w, http.StatusOK, dto.NewEnterpriseDto(enterprise.Info()))
}

func (h *GameHandler) completeGame(w http.ResponseWriter, r *http.Request) {
	enterprise := h.getEnterprise(w)
	if enterprise == nil {
		return
	}
	info, err := enterprise.Complete()
	if err != nil {
		writeGameError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, dto.NewEnterpriseDto(info))
	h.doneOnce.Do(func() { close(h.done) })
}

func (h *GameHandler) startGame(w http.ResponseWriter, r *http.Request) {
	h.mu.Lock()
	if h.closed || h.enterprise != nil {
		h.mu.Unlock()
		writeError(w, http.StatusConflict, "Предприятие уже запущено или игра завершена")
		return
	}
	h.enterprise = game.NewEnterprise()
	info := h.enterprise.Info()
	h.mu.Unlock()
	writeJSON(w, http.StatusCreated, dto.NewEnterpriseDto(info))
}
