package http

import (
	"MiningGame/game/miners"
	"MiningGame/http/dto"
	"net/http"
)

func (h *GameHandler) getAllMinerTypes(w http.ResponseWriter, r *http.Request) {
	configs := miners.GetMinerTypes()
	response := make([]dto.MinerDto, 0, len(configs))
	for _, config := range configs {
		response = append(response, dto.MinerDto(config))
	}
	writeJSON(w, http.StatusOK, response)
}

func (h *GameHandler) hireNewMiner(w http.ResponseWriter, r *http.Request) {
	enterprise := h.getEnterprise(w)
	if enterprise == nil {
		return
	}
	request, ok := decodeRequest[dto.MinerTypeDto](w, r)
	if !ok {
		return
	}
	miner, err := enterprise.HireNewMiner(request.MinerType)
	if err != nil {
		writeGameError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, dto.NewMinerStateDto(miner))
}

func (h *GameHandler) getMiners(w http.ResponseWriter, r *http.Request) {
	enterprise := h.getEnterprise(w)
	if enterprise == nil {
		return
	}
	query := r.URL.Query()
	filter := miners.MinerType(query.Get("type"))
	if values, ok := query["type"]; ok && (len(values) != 1 || filter == "") {
		writeError(w, http.StatusBadRequest, "Укажите один допустимый класс шахтёра")
		return
	}
	workers, err := enterprise.Miners(filter)
	if err != nil {
		writeGameError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, dto.NewMinerStatesDto(workers))
}
