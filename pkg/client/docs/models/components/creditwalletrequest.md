# CreditWalletRequest


## Fields

| Field                                                      | Type                                                       | Required                                                   | Description                                                |
| ---------------------------------------------------------- | ---------------------------------------------------------- | ---------------------------------------------------------- | ---------------------------------------------------------- |
| `Amount`                                                   | [components.Monetary](../../models/components/monetary.md) | :heavy_check_mark:                                         | N/A                                                        |
| `Metadata`                                                 | map[string]*string*                                        | :heavy_minus_sign:                                         | Metadata associated with the wallet.                       |
| `Reference`                                                | **string*                                                  | :heavy_minus_sign:                                         | N/A                                                        |
| `Sources`                                                  | [][components.Subject](../../models/components/subject.md) | :heavy_minus_sign:                                         | N/A                                                        |
| `Balance`                                                  | **string*                                                  | :heavy_minus_sign:                                         | The balance to credit                                      |
| `Timestamp`                                                | [*time.Time](https://pkg.go.dev/time#Time)                 | :heavy_minus_sign:                                         | N/A                                                        |