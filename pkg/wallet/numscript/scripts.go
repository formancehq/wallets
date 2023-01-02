//nolint:golint
package numscript

import (
	"bytes"
	_ "embed"
	"text/template"
)

var (
	//go:embed confirm-hold.num
	ConfirmHold string
	//go:embed cancel-hold.num
	CancelHold string
	//go:embed credit-wallet.num
	CreditWallet string
	//go:embed debit-wallet.num
	DebitWallet string
)

func renderTemplate(tplStr string, data any) string {
	buf := bytes.NewBufferString("")
	tpl := template.Must(template.New("tpl").Parse(tplStr))
	if err := tpl.Execute(buf, data); err != nil {
		panic(err)
	}
	return buf.String()
}

func BuildConfirmHoldScript(final bool, asset string) string {
	return renderTemplate(ConfirmHold, map[string]any{
		"Final": final,
		"Asset": asset,
	})
}

func BuildCreditWalletScript() string {
	return renderTemplate(CreditWallet, map[string]any{})
}

func BuildDebitWalletScript() string {
	return renderTemplate(DebitWallet, map[string]any{})
}
