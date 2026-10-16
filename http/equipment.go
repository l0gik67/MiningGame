package http

import (
	"MiningGame/http/dto"
	"net/http"
)

func (h *GameHandler) getEquipment(w http.ResponseWriter, r *http.Request) {
	enterprise := h.getEnterprise(w)
	if enterprise == nil {
		return
	}
	writeJSON(w, http.StatusOK, dto.NewEquipmentDtos(enterprise.Equipment()))
}

func (h *GameHandler) buyEquipment(w http.ResponseWriter, r *http.Request) {
	enterprise := h.getEnterprise(w)
	if enterprise == nil {
		return
	}
	request, ok := decodeRequest[dto.EquipmentTypeDto](w, r)
	if !ok {
		return
	}
	equipment, err := enterprise.BuyEquipment(request.Type)
	if err != nil {
		writeGameError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, dto.EquipmentDto(equipment))
}
