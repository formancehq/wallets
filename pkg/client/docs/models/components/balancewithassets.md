# BalanceWithAssets


## Fields

| Field                                                  | Type                                                   | Required                                               | Description                                            |
| ------------------------------------------------------ | ------------------------------------------------------ | ------------------------------------------------------ | ------------------------------------------------------ |
| `Name`                                                 | *string*                                               | :heavy_check_mark:                                     | N/A                                                    |
| `ExpiresAt`                                            | [*time.Time](https://pkg.go.dev/time#Time)             | :heavy_minus_sign:                                     | N/A                                                    |
| `Priority`                                             | [*big.Int](https://pkg.go.dev/math/big#Int)            | :heavy_minus_sign:                                     | N/A                                                    |
| `Assets`                                               | map[string][*big.Int](https://pkg.go.dev/math/big#Int) | :heavy_check_mark:                                     | N/A                                                    |