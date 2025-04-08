# Hold


## Fields

| Field                                                     | Type                                                      | Required                                                  | Description                                               |
| --------------------------------------------------------- | --------------------------------------------------------- | --------------------------------------------------------- | --------------------------------------------------------- |
| `ID`                                                      | *string*                                                  | :heavy_check_mark:                                        | The unique ID of the hold.                                |
| `WalletID`                                                | *string*                                                  | :heavy_check_mark:                                        | The ID of the wallet the hold is associated with.         |
| `Metadata`                                                | map[string]*string*                                       | :heavy_check_mark:                                        | Metadata associated with the hold.                        |
| `Asset`                                                   | **string*                                                 | :heavy_minus_sign:                                        | N/A                                                       |
| `Description`                                             | *string*                                                  | :heavy_check_mark:                                        | N/A                                                       |
| `Destination`                                             | [*components.Subject](../../models/components/subject.md) | :heavy_minus_sign:                                        | N/A                                                       |