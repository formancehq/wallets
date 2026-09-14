# Wallet

A wallet backed by a set of ledger accounts


## Fields

| Field                                                       | Type                                                        | Required                                                    | Description                                                 |
| ----------------------------------------------------------- | ----------------------------------------------------------- | ----------------------------------------------------------- | ----------------------------------------------------------- |
| `ID`                                                        | *string*                                                    | :heavy_check_mark:                                          | The unique ID of the wallet.                                |
| `Metadata`                                                  | map[string]*string*                                         | :heavy_check_mark:                                          | Metadata associated with the wallet.                        |
| `Name`                                                      | *string*                                                    | :heavy_check_mark:                                          | Human-readable name of the wallet                           |
| `CreatedAt`                                                 | [time.Time](https://pkg.go.dev/time#Time)                   | :heavy_check_mark:                                          | When the wallet was created                                 |
| `Ledger`                                                    | *string*                                                    | :heavy_check_mark:                                          | Name of the ledger backing this wallet                      |
| `Balances`                                                  | [*components.Balances](../../models/components/balances.md) | :heavy_minus_sign:                                          | The wallet's main balance, keyed by asset                   |