# LedgerAccountSubject


## Fields

| Field                                                      | Type                                                       | Required                                                   | Description                                                |
| ---------------------------------------------------------- | ---------------------------------------------------------- | ---------------------------------------------------------- | ---------------------------------------------------------- |
| `Type`                                                     | *string*                                                   | :heavy_check_mark:                                         | Discriminator identifying this subject as a ledger account |
| `Identifier`                                               | *string*                                                   | :heavy_check_mark:                                         | Address of the ledger account                              |