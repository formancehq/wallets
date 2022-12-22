//nolint:golint
package numscript

import (
	_ "embed"
)

var (
	//go:embed confirm-hold.num
	ConfirmHold string
	//go:embed cancel-hold.num
	CancelHold string
)
