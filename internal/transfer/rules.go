package transfer

import "errors"

// Bank is the sentinel from/to value meaning "the session's Bank account".
// Player IDs are 32-char hex, so they can never collide with this.
const Bank = "bank"

var (
	ErrBadAmount     = errors.New("transfer: amount must be a positive whole dollar")
	ErrBadCode       = errors.New("transfer: unknown transfer code")
	ErrBadDirection  = errors.New("transfer: transfer code does not match from/to")
	ErrSelfTransfer  = errors.New("transfer: cannot transfer to self")
	ErrUnknownPlayer = errors.New("transfer: no such player in session")
	ErrNotStarted    = errors.New("transfer: game has not started")
)

// direction says which side of transfer code must be the Bank.
type direction struct{ fromBank, toBank bool }

// codeDirections is the allowed transfer codes and the direction each implies.
// Admin code 99 is deliberately absent -- it's not reachable via player-initiated
// transfers, only the admin slice.
var codeDirections = map[uint16]direction{
	10: {fromBank: false, toBank: false}, // Player -> Player
	20: {fromBank: true, toBank: false},  // Bank -> Player (GO salary)
	21: {fromBank: true, toBank: false},  // Bank -> Player (mortgage)
	22: {fromBank: true, toBank: false},  // Bank -> Player (chance/chest)
	23: {fromBank: true, toBank: false},  // Bank -> Player (generic)
	30: {fromBank: false, toBank: true},  // Player -> Bank (property)
	31: {fromBank: false, toBank: true},  // Player -> Bank (tax)
	32: {fromBank: false, toBank: true},  // Player -> Bank (generic)
}

// validate checks amount/code/direction from the from/to sentinels alone, before
// any DB hit -- bad input fails fast. Player existence is checked later, at
// account resolution.
func validate(from, to string, amount uint64, code uint16) error {
	if amount == 0 {
		return ErrBadAmount // uint64 is already whole + non-negative; 0 is the only bad value
	}
	dir, ok := codeDirections[code]
	if !ok {
		return ErrBadCode
	}
	if from == to {
		return ErrSelfTransfer
	}
	if (from == Bank) != dir.fromBank || (to == Bank) != dir.toBank {
		return ErrBadDirection
	}
	return nil
}
