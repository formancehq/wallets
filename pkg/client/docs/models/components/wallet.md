# Wallet


## Fields

| Field                                                       | Type                                                        | Required                                                    | Description                                                 |
| ----------------------------------------------------------- | ----------------------------------------------------------- | ----------------------------------------------------------- | ----------------------------------------------------------- |
| `ID`                                                        | *string*                                                    | :heavy_check_mark:                                          | The unique ID of the wallet.                                |
| `Metadata`                                                  | map[string]*string*                                         | :heavy_check_mark:                                          | Metadata associated with the wallet.                        |
| `Name`                                                      | *string*                                                    | :heavy_check_mark:                                          | N/A                                                         |
| `CreatedAt`                                                 | [time.Time](https://pkg.go.dev/time#Time)                   | :heavy_check_mark:                                          | N/A                                                         |
| `Ledger`                                                    | *string*                                                    | :heavy_check_mark:                                          | N/A                                                         |
| `Balances`                                                  | [*components.Balances](../../models/components/balances.md) | :heavy_minus_sign:                                          | N/A                                                         |