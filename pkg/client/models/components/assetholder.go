// Code generated by Speakeasy (https://speakeasy.com). DO NOT EDIT.

package components

import (
	"github.com/formancehq/wallets/pkg/client/internal/utils"
	"math/big"
)

type AssetHolder struct {
	Assets map[string]*big.Int `json:"assets"`
}

func (a AssetHolder) MarshalJSON() ([]byte, error) {
	return utils.MarshalJSON(a, "", false)
}

func (a *AssetHolder) UnmarshalJSON(data []byte) error {
	if err := utils.UnmarshalJSON(data, &a, "", false, false); err != nil {
		return err
	}
	return nil
}

func (o *AssetHolder) GetAssets() map[string]*big.Int {
	if o == nil {
		return map[string]*big.Int{}
	}
	return o.Assets
}
