# Hold

Funds locked by a pending debit, later either confirmed or voided


## Fields

| Field                                                                            | Type                                                                             | Required                                                                         | Description                                                                      |
| -------------------------------------------------------------------------------- | -------------------------------------------------------------------------------- | -------------------------------------------------------------------------------- | -------------------------------------------------------------------------------- |
| `ID`                                                                             | *string*                                                                         | :heavy_check_mark:                                                               | The unique ID of the hold.                                                       |
| `WalletID`                                                                       | *string*                                                                         | :heavy_check_mark:                                                               | The ID of the wallet the hold is associated with.                                |
| `Metadata`                                                                       | map[string]*string*                                                              | :heavy_check_mark:                                                               | Metadata associated with the hold.                                               |
| `Asset`                                                                          | *string*                                                                         | :heavy_check_mark:                                                               | Asset the held funds are denominated in                                          |
| `Description`                                                                    | *string*                                                                         | :heavy_check_mark:                                                               | Human-readable reason the funds were held                                        |
| `Destination`                                                                    | [*components.Subject](../../models/components/subject.md)                        | :heavy_minus_sign:                                                               | The counterparty of a wallet movement, either a ledger account or another wallet |