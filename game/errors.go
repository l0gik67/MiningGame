package game

import "errors"

var (
	ErrGameStopped               = errors.New("game has stopped")
	ErrUnknownEquipment          = errors.New("unknown equipment type")
	ErrEquipmentAlreadyPurchased = errors.New("equipment already purchased")
	ErrNotEnoughCoal             = errors.New("not enough coal")
	ErrEquipmentIncomplete       = errors.New("all equipment must be purchased before completing the game")
)
