# V1
(*Wallets.V1*)

### Available Operations

* [GetServerInfo](#getserverinfo) - Get server info
* [GetTransactions](#gettransactions)
* [ListWallets](#listwallets) - List all wallets
* [CreateWallet](#createwallet) - Create a new wallet
* [GetWallet](#getwallet) - Get a wallet
* [UpdateWallet](#updatewallet) - Update a wallet
* [GetWalletSummary](#getwalletsummary) - Get wallet summary
* [ListBalances](#listbalances) - List balances of a wallet
* [CreateBalance](#createbalance) - Create a balance
* [GetBalance](#getbalance) - Get detailed balance
* [DebitWallet](#debitwallet) - Debit a wallet
* [CreditWallet](#creditwallet) - Credit a wallet
* [GetHolds](#getholds) - Get all holds for a wallet
* [GetHold](#gethold) - Get a hold
* [ConfirmHold](#confirmhold) - Confirm a hold
* [VoidHold](#voidhold) - Cancel a hold

## GetServerInfo

Get server info

### Example Usage

```go
package main

import(
	"openapi/models/components"
	"openapi"
	"context"
	"log"
)

func main() {
    s := openapi.New(
        openapi.WithSecurity(components.Security{
            ClientID: "",
            ClientSecret: "",
        }),
    )

    ctx := context.Background()
    res, err := s.Wallets.V1.GetServerInfo(ctx)
    if err != nil {
        log.Fatal(err)
    }
    if res.ServerInfo != nil {
        // handle response
    }
}
```

### Parameters

| Parameter                                                | Type                                                     | Required                                                 | Description                                              |
| -------------------------------------------------------- | -------------------------------------------------------- | -------------------------------------------------------- | -------------------------------------------------------- |
| `ctx`                                                    | [context.Context](https://pkg.go.dev/context#Context)    | :heavy_check_mark:                                       | The context to use for the request.                      |
| `opts`                                                   | [][operations.Option](../../models/operations/option.md) | :heavy_minus_sign:                                       | The options for this request.                            |


### Response

**[*operations.GetServerInfoResponse](../../models/operations/getserverinforesponse.md), error**
| Error Object       | Status Code        | Content Type       |
| ------------------ | ------------------ | ------------------ |
| sdkerrors.SDKError | 4xx-5xx            | */*                |

## GetTransactions

### Example Usage

```go
package main

import(
	"openapi/models/components"
	"openapi"
	"context"
	"log"
)

func main() {
    s := openapi.New(
        openapi.WithSecurity(components.Security{
            ClientID: "",
            ClientSecret: "",
        }),
    )
    var pageSize *int64 = openapi.Int64(100)

    var walletID *string = openapi.String("wallet1")

    var cursor *string = openapi.String("aHR0cHM6Ly9nLnBhZ2UvTmVrby1SYW1lbj9zaGFyZQ==")
    ctx := context.Background()
    res, err := s.Wallets.V1.GetTransactions(ctx, pageSize, walletID, cursor)
    if err != nil {
        log.Fatal(err)
    }
    if res.GetTransactionsResponse != nil {
        // handle response
    }
}
```

### Parameters

| Parameter                                                                                                                                                                                                            | Type                                                                                                                                                                                                                 | Required                                                                                                                                                                                                             | Description                                                                                                                                                                                                          | Example                                                                                                                                                                                                              |
| -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `ctx`                                                                                                                                                                                                                | [context.Context](https://pkg.go.dev/context#Context)                                                                                                                                                                | :heavy_check_mark:                                                                                                                                                                                                   | The context to use for the request.                                                                                                                                                                                  |                                                                                                                                                                                                                      |
| `pageSize`                                                                                                                                                                                                           | **int64*                                                                                                                                                                                                             | :heavy_minus_sign:                                                                                                                                                                                                   | The maximum number of results to return per page                                                                                                                                                                     | 100                                                                                                                                                                                                                  |
| `walletID`                                                                                                                                                                                                           | **string*                                                                                                                                                                                                            | :heavy_minus_sign:                                                                                                                                                                                                   | A wallet ID to filter on                                                                                                                                                                                             | wallet1                                                                                                                                                                                                              |
| `cursor`                                                                                                                                                                                                             | **string*                                                                                                                                                                                                            | :heavy_minus_sign:                                                                                                                                                                                                   | Parameter used in pagination requests.<br/>Set to the value of next for the next page of results.<br/>Set to the value of previous for the previous page of results.<br/>No other parameters can be set when the cursor is set.<br/> | aHR0cHM6Ly9nLnBhZ2UvTmVrby1SYW1lbj9zaGFyZQ==                                                                                                                                                                         |
| `opts`                                                                                                                                                                                                               | [][operations.Option](../../models/operations/option.md)                                                                                                                                                             | :heavy_minus_sign:                                                                                                                                                                                                   | The options for this request.                                                                                                                                                                                        |                                                                                                                                                                                                                      |


### Response

**[*operations.GetTransactionsResponse](../../models/operations/gettransactionsresponse.md), error**
| Error Object       | Status Code        | Content Type       |
| ------------------ | ------------------ | ------------------ |
| sdkerrors.SDKError | 4xx-5xx            | */*                |

## ListWallets

List all wallets

### Example Usage

```go
package main

import(
	"openapi/models/components"
	"openapi"
	"openapi/models/operations"
	"context"
	"log"
)

func main() {
    s := openapi.New(
        openapi.WithSecurity(components.Security{
            ClientID: "",
            ClientSecret: "",
        }),
    )
    request := operations.ListWalletsRequest{
        Name: openapi.String("wallet1"),
        Metadata: map[string]string{
            "admin": "true",
        },
        PageSize: openapi.Int64(100),
        Cursor: openapi.String("aHR0cHM6Ly9nLnBhZ2UvTmVrby1SYW1lbj9zaGFyZQ=="),
        Expand: openapi.String("balances"),
    }
    ctx := context.Background()
    res, err := s.Wallets.V1.ListWallets(ctx, request)
    if err != nil {
        log.Fatal(err)
    }
    if res.ListWalletsResponse != nil {
        // handle response
    }
}
```

### Parameters

| Parameter                                                                      | Type                                                                           | Required                                                                       | Description                                                                    |
| ------------------------------------------------------------------------------ | ------------------------------------------------------------------------------ | ------------------------------------------------------------------------------ | ------------------------------------------------------------------------------ |
| `ctx`                                                                          | [context.Context](https://pkg.go.dev/context#Context)                          | :heavy_check_mark:                                                             | The context to use for the request.                                            |
| `request`                                                                      | [operations.ListWalletsRequest](../../models/operations/listwalletsrequest.md) | :heavy_check_mark:                                                             | The request object to use for the request.                                     |
| `opts`                                                                         | [][operations.Option](../../models/operations/option.md)                       | :heavy_minus_sign:                                                             | The options for this request.                                                  |


### Response

**[*operations.ListWalletsResponse](../../models/operations/listwalletsresponse.md), error**
| Error Object       | Status Code        | Content Type       |
| ------------------ | ------------------ | ------------------ |
| sdkerrors.SDKError | 4xx-5xx            | */*                |

## CreateWallet

Create a new wallet

### Example Usage

```go
package main

import(
	"openapi/models/components"
	"openapi"
	"context"
	"log"
)

func main() {
    s := openapi.New(
        openapi.WithSecurity(components.Security{
            ClientID: "",
            ClientSecret: "",
        }),
    )

    ctx := context.Background()
    res, err := s.Wallets.V1.CreateWallet(ctx, nil, nil)
    if err != nil {
        log.Fatal(err)
    }
    if res.CreateWalletResponse != nil {
        // handle response
    }
}
```

### Parameters

| Parameter                                                                         | Type                                                                              | Required                                                                          | Description                                                                       |
| --------------------------------------------------------------------------------- | --------------------------------------------------------------------------------- | --------------------------------------------------------------------------------- | --------------------------------------------------------------------------------- |
| `ctx`                                                                             | [context.Context](https://pkg.go.dev/context#Context)                             | :heavy_check_mark:                                                                | The context to use for the request.                                               |
| `idempotencyKey`                                                                  | **string*                                                                         | :heavy_minus_sign:                                                                | Use an idempotency key                                                            |
| `createWalletRequest`                                                             | [*components.CreateWalletRequest](../../models/components/createwalletrequest.md) | :heavy_minus_sign:                                                                | N/A                                                                               |
| `opts`                                                                            | [][operations.Option](../../models/operations/option.md)                          | :heavy_minus_sign:                                                                | The options for this request.                                                     |


### Response

**[*operations.CreateWalletResponse](../../models/operations/createwalletresponse.md), error**
| Error Object       | Status Code        | Content Type       |
| ------------------ | ------------------ | ------------------ |
| sdkerrors.SDKError | 4xx-5xx            | */*                |

## GetWallet

Get a wallet

### Example Usage

```go
package main

import(
	"openapi/models/components"
	"openapi"
	"context"
	"log"
)

func main() {
    s := openapi.New(
        openapi.WithSecurity(components.Security{
            ClientID: "",
            ClientSecret: "",
        }),
    )
    var id string = "<value>"
    ctx := context.Background()
    res, err := s.Wallets.V1.GetWallet(ctx, id)
    if err != nil {
        log.Fatal(err)
    }
    if res.GetWalletResponse != nil {
        // handle response
    }
}
```

### Parameters

| Parameter                                                | Type                                                     | Required                                                 | Description                                              |
| -------------------------------------------------------- | -------------------------------------------------------- | -------------------------------------------------------- | -------------------------------------------------------- |
| `ctx`                                                    | [context.Context](https://pkg.go.dev/context#Context)    | :heavy_check_mark:                                       | The context to use for the request.                      |
| `id`                                                     | *string*                                                 | :heavy_check_mark:                                       | N/A                                                      |
| `opts`                                                   | [][operations.Option](../../models/operations/option.md) | :heavy_minus_sign:                                       | The options for this request.                            |


### Response

**[*operations.GetWalletResponse](../../models/operations/getwalletresponse.md), error**
| Error Object       | Status Code        | Content Type       |
| ------------------ | ------------------ | ------------------ |
| sdkerrors.SDKError | 4xx-5xx            | */*                |

## UpdateWallet

Update a wallet

### Example Usage

```go
package main

import(
	"openapi/models/components"
	"openapi"
	"openapi/models/operations"
	"context"
	"log"
)

func main() {
    s := openapi.New(
        openapi.WithSecurity(components.Security{
            ClientID: "",
            ClientSecret: "",
        }),
    )
    var id string = "<value>"
    ctx := context.Background()
    res, err := s.Wallets.V1.UpdateWallet(ctx, id, nil, nil)
    if err != nil {
        log.Fatal(err)
    }
    if res != nil {
        // handle response
    }
}
```

### Parameters

| Parameter                                                                                 | Type                                                                                      | Required                                                                                  | Description                                                                               |
| ----------------------------------------------------------------------------------------- | ----------------------------------------------------------------------------------------- | ----------------------------------------------------------------------------------------- | ----------------------------------------------------------------------------------------- |
| `ctx`                                                                                     | [context.Context](https://pkg.go.dev/context#Context)                                     | :heavy_check_mark:                                                                        | The context to use for the request.                                                       |
| `id`                                                                                      | *string*                                                                                  | :heavy_check_mark:                                                                        | N/A                                                                                       |
| `idempotencyKey`                                                                          | **string*                                                                                 | :heavy_minus_sign:                                                                        | Use an idempotency key                                                                    |
| `requestBody`                                                                             | [*operations.UpdateWalletRequestBody](../../models/operations/updatewalletrequestbody.md) | :heavy_minus_sign:                                                                        | N/A                                                                                       |
| `opts`                                                                                    | [][operations.Option](../../models/operations/option.md)                                  | :heavy_minus_sign:                                                                        | The options for this request.                                                             |


### Response

**[*operations.UpdateWalletResponse](../../models/operations/updatewalletresponse.md), error**
| Error Object       | Status Code        | Content Type       |
| ------------------ | ------------------ | ------------------ |
| sdkerrors.SDKError | 4xx-5xx            | */*                |

## GetWalletSummary

Get wallet summary

### Example Usage

```go
package main

import(
	"openapi/models/components"
	"openapi"
	"context"
	"log"
)

func main() {
    s := openapi.New(
        openapi.WithSecurity(components.Security{
            ClientID: "",
            ClientSecret: "",
        }),
    )
    var id string = "<value>"
    ctx := context.Background()
    res, err := s.Wallets.V1.GetWalletSummary(ctx, id)
    if err != nil {
        log.Fatal(err)
    }
    if res.GetWalletSummaryResponse != nil {
        // handle response
    }
}
```

### Parameters

| Parameter                                                | Type                                                     | Required                                                 | Description                                              |
| -------------------------------------------------------- | -------------------------------------------------------- | -------------------------------------------------------- | -------------------------------------------------------- |
| `ctx`                                                    | [context.Context](https://pkg.go.dev/context#Context)    | :heavy_check_mark:                                       | The context to use for the request.                      |
| `id`                                                     | *string*                                                 | :heavy_check_mark:                                       | N/A                                                      |
| `opts`                                                   | [][operations.Option](../../models/operations/option.md) | :heavy_minus_sign:                                       | The options for this request.                            |


### Response

**[*operations.GetWalletSummaryResponse](../../models/operations/getwalletsummaryresponse.md), error**
| Error Object       | Status Code        | Content Type       |
| ------------------ | ------------------ | ------------------ |
| sdkerrors.SDKError | 4xx-5xx            | */*                |

## ListBalances

List balances of a wallet

### Example Usage

```go
package main

import(
	"openapi/models/components"
	"openapi"
	"context"
	"log"
)

func main() {
    s := openapi.New(
        openapi.WithSecurity(components.Security{
            ClientID: "",
            ClientSecret: "",
        }),
    )
    var id string = "<value>"
    ctx := context.Background()
    res, err := s.Wallets.V1.ListBalances(ctx, id)
    if err != nil {
        log.Fatal(err)
    }
    if res.ListBalancesResponse != nil {
        // handle response
    }
}
```

### Parameters

| Parameter                                                | Type                                                     | Required                                                 | Description                                              |
| -------------------------------------------------------- | -------------------------------------------------------- | -------------------------------------------------------- | -------------------------------------------------------- |
| `ctx`                                                    | [context.Context](https://pkg.go.dev/context#Context)    | :heavy_check_mark:                                       | The context to use for the request.                      |
| `id`                                                     | *string*                                                 | :heavy_check_mark:                                       | N/A                                                      |
| `opts`                                                   | [][operations.Option](../../models/operations/option.md) | :heavy_minus_sign:                                       | The options for this request.                            |


### Response

**[*operations.ListBalancesResponse](../../models/operations/listbalancesresponse.md), error**
| Error Object       | Status Code        | Content Type       |
| ------------------ | ------------------ | ------------------ |
| sdkerrors.SDKError | 4xx-5xx            | */*                |

## CreateBalance

Create a balance

### Example Usage

```go
package main

import(
	"openapi/models/components"
	"openapi"
	"context"
	"log"
)

func main() {
    s := openapi.New(
        openapi.WithSecurity(components.Security{
            ClientID: "",
            ClientSecret: "",
        }),
    )
    var id string = "<value>"
    ctx := context.Background()
    res, err := s.Wallets.V1.CreateBalance(ctx, id, nil, nil)
    if err != nil {
        log.Fatal(err)
    }
    if res.CreateBalanceResponse != nil {
        // handle response
    }
}
```

### Parameters

| Parameter                                                                           | Type                                                                                | Required                                                                            | Description                                                                         |
| ----------------------------------------------------------------------------------- | ----------------------------------------------------------------------------------- | ----------------------------------------------------------------------------------- | ----------------------------------------------------------------------------------- |
| `ctx`                                                                               | [context.Context](https://pkg.go.dev/context#Context)                               | :heavy_check_mark:                                                                  | The context to use for the request.                                                 |
| `id`                                                                                | *string*                                                                            | :heavy_check_mark:                                                                  | N/A                                                                                 |
| `idempotencyKey`                                                                    | **string*                                                                           | :heavy_minus_sign:                                                                  | Use an idempotency key                                                              |
| `createBalanceRequest`                                                              | [*components.CreateBalanceRequest](../../models/components/createbalancerequest.md) | :heavy_minus_sign:                                                                  | N/A                                                                                 |
| `opts`                                                                              | [][operations.Option](../../models/operations/option.md)                            | :heavy_minus_sign:                                                                  | The options for this request.                                                       |


### Response

**[*operations.CreateBalanceResponse](../../models/operations/createbalanceresponse.md), error**
| Error Object       | Status Code        | Content Type       |
| ------------------ | ------------------ | ------------------ |
| sdkerrors.SDKError | 4xx-5xx            | */*                |

## GetBalance

Get detailed balance

### Example Usage

```go
package main

import(
	"openapi/models/components"
	"openapi"
	"context"
	"log"
)

func main() {
    s := openapi.New(
        openapi.WithSecurity(components.Security{
            ClientID: "",
            ClientSecret: "",
        }),
    )
    var id string = "<value>"

    var balanceName string = "<value>"
    ctx := context.Background()
    res, err := s.Wallets.V1.GetBalance(ctx, id, balanceName)
    if err != nil {
        log.Fatal(err)
    }
    if res.GetBalanceResponse != nil {
        // handle response
    }
}
```

### Parameters

| Parameter                                                | Type                                                     | Required                                                 | Description                                              |
| -------------------------------------------------------- | -------------------------------------------------------- | -------------------------------------------------------- | -------------------------------------------------------- |
| `ctx`                                                    | [context.Context](https://pkg.go.dev/context#Context)    | :heavy_check_mark:                                       | The context to use for the request.                      |
| `id`                                                     | *string*                                                 | :heavy_check_mark:                                       | N/A                                                      |
| `balanceName`                                            | *string*                                                 | :heavy_check_mark:                                       | N/A                                                      |
| `opts`                                                   | [][operations.Option](../../models/operations/option.md) | :heavy_minus_sign:                                       | The options for this request.                            |


### Response

**[*operations.GetBalanceResponse](../../models/operations/getbalanceresponse.md), error**
| Error Object       | Status Code        | Content Type       |
| ------------------ | ------------------ | ------------------ |
| sdkerrors.SDKError | 4xx-5xx            | */*                |

## DebitWallet

Debit a wallet

### Example Usage

```go
package main

import(
	"openapi/models/components"
	"openapi"
	"math/big"
	"context"
	"log"
)

func main() {
    s := openapi.New(
        openapi.WithSecurity(components.Security{
            ClientID: "",
            ClientSecret: "",
        }),
    )
    var id string = "<value>"

    var debitWalletRequest *components.DebitWalletRequest = &components.DebitWalletRequest{
        Amount: components.Monetary{
            Asset: "USD/2",
            Amount: big.NewInt(100),
        },
        Pending: openapi.Bool(true),
        Metadata: map[string]string{
            "key": "",
        },
    }
    ctx := context.Background()
    res, err := s.Wallets.V1.DebitWallet(ctx, id, nil, debitWalletRequest)
    if err != nil {
        log.Fatal(err)
    }
    if res.DebitWalletResponse != nil {
        // handle response
    }
}
```

### Parameters

| Parameter                                                                                     | Type                                                                                          | Required                                                                                      | Description                                                                                   | Example                                                                                       |
| --------------------------------------------------------------------------------------------- | --------------------------------------------------------------------------------------------- | --------------------------------------------------------------------------------------------- | --------------------------------------------------------------------------------------------- | --------------------------------------------------------------------------------------------- |
| `ctx`                                                                                         | [context.Context](https://pkg.go.dev/context#Context)                                         | :heavy_check_mark:                                                                            | The context to use for the request.                                                           |                                                                                               |
| `id`                                                                                          | *string*                                                                                      | :heavy_check_mark:                                                                            | N/A                                                                                           |                                                                                               |
| `idempotencyKey`                                                                              | **string*                                                                                     | :heavy_minus_sign:                                                                            | Use an idempotency key                                                                        |                                                                                               |
| `debitWalletRequest`                                                                          | [*components.DebitWalletRequest](../../models/components/debitwalletrequest.md)               | :heavy_minus_sign:                                                                            | N/A                                                                                           | {<br/>"amount": {<br/>"asset": "USD/2",<br/>"amount": 100<br/>},<br/>"metadata": {<br/>"key": ""<br/>},<br/>"pending": true<br/>} |
| `opts`                                                                                        | [][operations.Option](../../models/operations/option.md)                                      | :heavy_minus_sign:                                                                            | The options for this request.                                                                 |                                                                                               |


### Response

**[*operations.DebitWalletResponse](../../models/operations/debitwalletresponse.md), error**
| Error Object       | Status Code        | Content Type       |
| ------------------ | ------------------ | ------------------ |
| sdkerrors.SDKError | 4xx-5xx            | */*                |

## CreditWallet

Credit a wallet

### Example Usage

```go
package main

import(
	"openapi/models/components"
	"openapi"
	"math/big"
	"context"
	"log"
)

func main() {
    s := openapi.New(
        openapi.WithSecurity(components.Security{
            ClientID: "",
            ClientSecret: "",
        }),
    )
    var id string = "<value>"

    var creditWalletRequest *components.CreditWalletRequest = &components.CreditWalletRequest{
        Amount: components.Monetary{
            Asset: "USD/2",
            Amount: big.NewInt(100),
        },
        Metadata: map[string]string{
            "key": "",
        },
        Sources: []components.Subject{
            components.CreateSubjectLedgerAccountSubject(
                components.LedgerAccountSubject{
                    Type: "<value>",
                    Identifier: "<value>",
                },
            ),
        },
    }
    ctx := context.Background()
    res, err := s.Wallets.V1.CreditWallet(ctx, id, nil, creditWalletRequest)
    if err != nil {
        log.Fatal(err)
    }
    if res != nil {
        // handle response
    }
}
```

### Parameters

| Parameter                                                                                   | Type                                                                                        | Required                                                                                    | Description                                                                                 | Example                                                                                     |
| ------------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------- |
| `ctx`                                                                                       | [context.Context](https://pkg.go.dev/context#Context)                                       | :heavy_check_mark:                                                                          | The context to use for the request.                                                         |                                                                                             |
| `id`                                                                                        | *string*                                                                                    | :heavy_check_mark:                                                                          | N/A                                                                                         |                                                                                             |
| `idempotencyKey`                                                                            | **string*                                                                                   | :heavy_minus_sign:                                                                          | Use an idempotency key                                                                      |                                                                                             |
| `creditWalletRequest`                                                                       | [*components.CreditWalletRequest](../../models/components/creditwalletrequest.md)           | :heavy_minus_sign:                                                                          | N/A                                                                                         | {<br/>"amount": {<br/>"asset": "USD/2",<br/>"amount": 100<br/>},<br/>"metadata": {<br/>"key": ""<br/>},<br/>"sources": []<br/>} |
| `opts`                                                                                      | [][operations.Option](../../models/operations/option.md)                                    | :heavy_minus_sign:                                                                          | The options for this request.                                                               |                                                                                             |


### Response

**[*operations.CreditWalletResponse](../../models/operations/creditwalletresponse.md), error**
| Error Object       | Status Code        | Content Type       |
| ------------------ | ------------------ | ------------------ |
| sdkerrors.SDKError | 4xx-5xx            | */*                |

## GetHolds

Get all holds for a wallet

### Example Usage

```go
package main

import(
	"openapi/models/components"
	"openapi"
	"context"
	"log"
)

func main() {
    s := openapi.New(
        openapi.WithSecurity(components.Security{
            ClientID: "",
            ClientSecret: "",
        }),
    )
    var pageSize *int64 = openapi.Int64(100)

    var walletID *string = openapi.String("wallet1")

    var metadata map[string]string = map[string]string{
        "admin": "true",
    }

    var cursor *string = openapi.String("aHR0cHM6Ly9nLnBhZ2UvTmVrby1SYW1lbj9zaGFyZQ==")
    ctx := context.Background()
    res, err := s.Wallets.V1.GetHolds(ctx, pageSize, walletID, metadata, cursor)
    if err != nil {
        log.Fatal(err)
    }
    if res.GetHoldsResponse != nil {
        // handle response
    }
}
```

### Parameters

| Parameter                                                                                                                                                                                                                      | Type                                                                                                                                                                                                                           | Required                                                                                                                                                                                                                       | Description                                                                                                                                                                                                                    | Example                                                                                                                                                                                                                        |
| ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| `ctx`                                                                                                                                                                                                                          | [context.Context](https://pkg.go.dev/context#Context)                                                                                                                                                                          | :heavy_check_mark:                                                                                                                                                                                                             | The context to use for the request.                                                                                                                                                                                            |                                                                                                                                                                                                                                |
| `pageSize`                                                                                                                                                                                                                     | **int64*                                                                                                                                                                                                                       | :heavy_minus_sign:                                                                                                                                                                                                             | The maximum number of results to return per page                                                                                                                                                                               | 100                                                                                                                                                                                                                            |
| `walletID`                                                                                                                                                                                                                     | **string*                                                                                                                                                                                                                      | :heavy_minus_sign:                                                                                                                                                                                                             | The wallet to filter on                                                                                                                                                                                                        | wallet1                                                                                                                                                                                                                        |
| `metadata`                                                                                                                                                                                                                     | map[string]*string*                                                                                                                                                                                                            | :heavy_minus_sign:                                                                                                                                                                                                             | Filter holds by metadata key value pairs. Nested objects can be used as seen in the example below.                                                                                                                             | {<br/>"admin": "true"<br/>}                                                                                                                                                                                                    |
| `cursor`                                                                                                                                                                                                                       | **string*                                                                                                                                                                                                                      | :heavy_minus_sign:                                                                                                                                                                                                             | Parameter used in pagination requests.<br/>Set to the value of next for the next page of results.<br/>Set to the value of previous for the previous page of results.<br/>No other parameters can be set when the pagination token is set.<br/> | aHR0cHM6Ly9nLnBhZ2UvTmVrby1SYW1lbj9zaGFyZQ==                                                                                                                                                                                   |
| `opts`                                                                                                                                                                                                                         | [][operations.Option](../../models/operations/option.md)                                                                                                                                                                       | :heavy_minus_sign:                                                                                                                                                                                                             | The options for this request.                                                                                                                                                                                                  |                                                                                                                                                                                                                                |


### Response

**[*operations.GetHoldsResponse](../../models/operations/getholdsresponse.md), error**
| Error Object       | Status Code        | Content Type       |
| ------------------ | ------------------ | ------------------ |
| sdkerrors.SDKError | 4xx-5xx            | */*                |

## GetHold

Get a hold

### Example Usage

```go
package main

import(
	"openapi/models/components"
	"openapi"
	"context"
	"log"
)

func main() {
    s := openapi.New(
        openapi.WithSecurity(components.Security{
            ClientID: "",
            ClientSecret: "",
        }),
    )
    var holdID string = "<value>"
    ctx := context.Background()
    res, err := s.Wallets.V1.GetHold(ctx, holdID)
    if err != nil {
        log.Fatal(err)
    }
    if res.GetHoldResponse != nil {
        // handle response
    }
}
```

### Parameters

| Parameter                                                | Type                                                     | Required                                                 | Description                                              |
| -------------------------------------------------------- | -------------------------------------------------------- | -------------------------------------------------------- | -------------------------------------------------------- |
| `ctx`                                                    | [context.Context](https://pkg.go.dev/context#Context)    | :heavy_check_mark:                                       | The context to use for the request.                      |
| `holdID`                                                 | *string*                                                 | :heavy_check_mark:                                       | The hold ID                                              |
| `opts`                                                   | [][operations.Option](../../models/operations/option.md) | :heavy_minus_sign:                                       | The options for this request.                            |


### Response

**[*operations.GetHoldResponse](../../models/operations/getholdresponse.md), error**
| Error Object       | Status Code        | Content Type       |
| ------------------ | ------------------ | ------------------ |
| sdkerrors.SDKError | 4xx-5xx            | */*                |

## ConfirmHold

Confirm a hold

### Example Usage

```go
package main

import(
	"openapi/models/components"
	"openapi"
	"math/big"
	"context"
	"log"
)

func main() {
    s := openapi.New(
        openapi.WithSecurity(components.Security{
            ClientID: "",
            ClientSecret: "",
        }),
    )
    var holdID string = "<value>"

    var confirmHoldRequest *components.ConfirmHoldRequest = &components.ConfirmHoldRequest{
        Amount: big.NewInt(100),
        Final: openapi.Bool(true),
    }
    ctx := context.Background()
    res, err := s.Wallets.V1.ConfirmHold(ctx, holdID, nil, confirmHoldRequest)
    if err != nil {
        log.Fatal(err)
    }
    if res != nil {
        // handle response
    }
}
```

### Parameters

| Parameter                                                                       | Type                                                                            | Required                                                                        | Description                                                                     |
| ------------------------------------------------------------------------------- | ------------------------------------------------------------------------------- | ------------------------------------------------------------------------------- | ------------------------------------------------------------------------------- |
| `ctx`                                                                           | [context.Context](https://pkg.go.dev/context#Context)                           | :heavy_check_mark:                                                              | The context to use for the request.                                             |
| `holdID`                                                                        | *string*                                                                        | :heavy_check_mark:                                                              | N/A                                                                             |
| `idempotencyKey`                                                                | **string*                                                                       | :heavy_minus_sign:                                                              | Use an idempotency key                                                          |
| `confirmHoldRequest`                                                            | [*components.ConfirmHoldRequest](../../models/components/confirmholdrequest.md) | :heavy_minus_sign:                                                              | N/A                                                                             |
| `opts`                                                                          | [][operations.Option](../../models/operations/option.md)                        | :heavy_minus_sign:                                                              | The options for this request.                                                   |


### Response

**[*operations.ConfirmHoldResponse](../../models/operations/confirmholdresponse.md), error**
| Error Object       | Status Code        | Content Type       |
| ------------------ | ------------------ | ------------------ |
| sdkerrors.SDKError | 4xx-5xx            | */*                |

## VoidHold

Cancel a hold

### Example Usage

```go
package main

import(
	"openapi/models/components"
	"openapi"
	"context"
	"log"
)

func main() {
    s := openapi.New(
        openapi.WithSecurity(components.Security{
            ClientID: "",
            ClientSecret: "",
        }),
    )
    var holdID string = "<value>"
    ctx := context.Background()
    res, err := s.Wallets.V1.VoidHold(ctx, holdID, nil)
    if err != nil {
        log.Fatal(err)
    }
    if res != nil {
        // handle response
    }
}
```

### Parameters

| Parameter                                                | Type                                                     | Required                                                 | Description                                              |
| -------------------------------------------------------- | -------------------------------------------------------- | -------------------------------------------------------- | -------------------------------------------------------- |
| `ctx`                                                    | [context.Context](https://pkg.go.dev/context#Context)    | :heavy_check_mark:                                       | The context to use for the request.                      |
| `holdID`                                                 | *string*                                                 | :heavy_check_mark:                                       | N/A                                                      |
| `idempotencyKey`                                         | **string*                                                | :heavy_minus_sign:                                       | Use an idempotency key                                   |
| `opts`                                                   | [][operations.Option](../../models/operations/option.md) | :heavy_minus_sign:                                       | The options for this request.                            |


### Response

**[*operations.VoidHoldResponse](../../models/operations/voidholdresponse.md), error**
| Error Object       | Status Code        | Content Type       |
| ------------------ | ------------------ | ------------------ |
| sdkerrors.SDKError | 4xx-5xx            | */*                |
