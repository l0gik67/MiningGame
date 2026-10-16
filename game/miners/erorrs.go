package miners

import "errors"

var NotEnoughMoneyForHiringError = errors.New("not enough money for hiring this type of worker")
var NoSuchMinerType = errors.New("no such miner type")
