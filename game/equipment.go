package game

import "time"

type EquipmentType string

const (
	EquipmentPickaxes    EquipmentType = "pickaxes"
	EquipmentVentilation EquipmentType = "ventilation"
	EquipmentCarts       EquipmentType = "carts"
)

type EquipmentInfo struct {
	Type      EquipmentType
	Cost      int
	Purchased bool
}

var equipmentCatalog = []EquipmentInfo{
	{Type: EquipmentPickaxes, Cost: 3000},
	{Type: EquipmentVentilation, Cost: 15000},
	{Type: EquipmentCarts, Cost: 50000},
}

func (e *Enterprise) Equipment() []EquipmentInfo {
	e.mutex.RLock()
	defer e.mutex.RUnlock()
	return e.equipmentLocked()
}

func (e *Enterprise) equipmentLocked() []EquipmentInfo {
	result := make([]EquipmentInfo, 0, len(equipmentCatalog))
	for _, equipment := range equipmentCatalog {
		equipment.Purchased = e.equipment[equipment.Type]
		result = append(result, equipment)
	}
	return result
}

func (e *Enterprise) BuyEquipment(equipmentType EquipmentType) (EquipmentInfo, error) {
	e.mutex.Lock()
	defer e.mutex.Unlock()
	if e.stopped {
		return EquipmentInfo{}, ErrGameStopped
	}
	for _, equipment := range equipmentCatalog {
		if equipment.Type != equipmentType {
			continue
		}
		if e.equipment[equipmentType] {
			return EquipmentInfo{}, ErrEquipmentAlreadyPurchased
		}
		if e.balance < equipment.Cost {
			return EquipmentInfo{}, ErrNotEnoughCoal
		}
		e.balance -= equipment.Cost
		e.equipment[equipmentType] = true
		allPurchased := true
		for _, required := range equipmentCatalog {
			if !e.equipment[required.Type] {
				allPurchased = false
				break
			}
		}
		if allPurchased {
			reached := time.Now()
			e.goalReachedAt = &reached
		}
		equipment.Purchased = true
		return equipment, nil
	}
	return EquipmentInfo{}, ErrUnknownEquipment
}
