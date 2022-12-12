package numscript

import (
	_ "embed"
)

var (
	//go:embed confirm-hold.num
	CancelHold string
	//go:embed confirm-hold.num
	ConfirmHold string
)
