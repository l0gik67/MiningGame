package dto

import (
	"MiningGame/game"
	"MiningGame/game/miners"
	"time"
)

type MinerStateDto struct {
	ID int `json:"id"`
	MinerDto
}

func NewMinerStateDto(state game.MinerState) MinerStateDto {
	return MinerStateDto{ID: state.ID, MinerDto: MinerDto(state.Info)}
}

func NewMinerStatesDto(states []game.MinerState) []MinerStateDto {
	result := make([]MinerStateDto, 0, len(states))
	for _, state := range states {
		result = append(result, NewMinerStateDto(state))
	}
	return result
}

type EquipmentTypeDto struct {
	Type game.EquipmentType `json:"type"`
}

type EquipmentDto struct {
	Type      game.EquipmentType `json:"type"`
	Cost      int                `json:"cost"`
	Purchased bool               `json:"purchased"`
}

func NewEquipmentDtos(equipment []game.EquipmentInfo) []EquipmentDto {
	result := make([]EquipmentDto, 0, len(equipment))
	for _, item := range equipment {
		result = append(result, EquipmentDto(item))
	}
	return result
}

type EnterpriseDto struct {
	Balance       int                      `json:"balance"`
	Miners        []MinerStateDto          `json:"miners"`
	Hired         map[miners.MinerType]int `json:"hired"`
	Equipment     []EquipmentDto           `json:"equipment"`
	StartedAt     time.Time                `json:"started_at"`
	FinishedAt    *time.Time               `json:"finished_at"`
	Duration      time.Duration            `json:"duration"`
	GoalReachedAt *time.Time               `json:"goal_reached_at"`
	GoalDuration  time.Duration            `json:"goal_duration"`
	Completed     bool                     `json:"completed"`
}

func NewEnterpriseDto(info game.EnterpriseInfo) EnterpriseDto {
	return EnterpriseDto{
		Balance:       info.Balance,
		Miners:        NewMinerStatesDto(info.Miners),
		Hired:         info.Hired,
		Equipment:     NewEquipmentDtos(info.Equipment),
		StartedAt:     info.StartedAt,
		FinishedAt:    info.FinishedAt,
		Duration:      info.Duration,
		GoalReachedAt: info.GoalReachedAt,
		GoalDuration:  info.GoalDuration,
		Completed:     info.Completed,
	}
}
