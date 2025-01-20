# openapi

<div align="left">
    <a href="https://www.speakeasy.com/?utm_source=<no value>&utm_campaign=go"><img src="https://custom-icon-badges.demolab.com/badge/-Built%20By%20Speakeasy-212015?style=for-the-badge&logoColor=FBE331&logo=speakeasy&labelColor=545454" /></a>
    <a href="https://opensource.org/licenses/MIT">
        <img src="https://img.shields.io/badge/License-MIT-blue.svg" style="width: 100px; height: 28px;" />
    </a>
</div>


## 🏗 **Welcome to your new SDK!** 🏗

It has been generated successfully based on your OpenAPI spec. However, it is not yet ready for production use. Here are some next steps:
- [ ] 🛠 Make your SDK feel handcrafted by [customizing it](https://www.speakeasy.com/docs/customize-sdks)
- [ ] ♻️ Refine your SDK quickly by iterating locally with the [Speakeasy CLI](https://github.com/speakeasy-api/speakeasy)
- [ ] 🎁 Publish your SDK to package managers by [configuring automatic publishing](https://www.speakeasy.com/docs/advanced-setup/publish-sdks)
- [ ] ✨ When ready to productionize, delete this section from the README

<!-- Start SDK Installation [installation] -->
## SDK Installation

```bash
go get openapi
```
<!-- End SDK Installation [installation] -->

<!-- Start SDK Example Usage [usage] -->
## SDK Example Usage

### Example

```go
package main

import (
	"context"
	"log"
	"openapi"
	"openapi/models/components"
)

func main() {
	s := openapi.New(
		openapi.WithSecurity(components.Security{
			ClientID:     "",
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
<!-- End SDK Example Usage [usage] -->

<!-- Start Available Resources and Operations [operations] -->
## Available Resources and Operations

### [Wallets.V1](docs/sdks/v1/README.md)

* [GetServerInfo](docs/sdks/v1/README.md#getserverinfo) - Get server info
* [GetTransactions](docs/sdks/v1/README.md#gettransactions)
* [ListWallets](docs/sdks/v1/README.md#listwallets) - List all wallets
* [CreateWallet](docs/sdks/v1/README.md#createwallet) - Create a new wallet
* [GetWallet](docs/sdks/v1/README.md#getwallet) - Get a wallet
* [UpdateWallet](docs/sdks/v1/README.md#updatewallet) - Update a wallet
* [GetWalletSummary](docs/sdks/v1/README.md#getwalletsummary) - Get wallet summary
* [ListBalances](docs/sdks/v1/README.md#listbalances) - List balances of a wallet
* [CreateBalance](docs/sdks/v1/README.md#createbalance) - Create a balance
* [GetBalance](docs/sdks/v1/README.md#getbalance) - Get detailed balance
* [DebitWallet](docs/sdks/v1/README.md#debitwallet) - Debit a wallet
* [CreditWallet](docs/sdks/v1/README.md#creditwallet) - Credit a wallet
* [GetHolds](docs/sdks/v1/README.md#getholds) - Get all holds for a wallet
* [GetHold](docs/sdks/v1/README.md#gethold) - Get a hold
* [ConfirmHold](docs/sdks/v1/README.md#confirmhold) - Confirm a hold
* [VoidHold](docs/sdks/v1/README.md#voidhold) - Cancel a hold
<!-- End Available Resources and Operations [operations] -->

<!-- Start Retries [retries] -->
## Retries

Some of the endpoints in this SDK support retries. If you use the SDK without any configuration, it will fall back to the default retry strategy provided by the API. However, the default retry strategy can be overridden on a per-operation basis, or across the entire SDK.

To change the default retry strategy for a single API call, simply provide a `retry.Config` object to the call by using the `WithRetries` option:
```go
package main

import (
	"context"
	"log"
	"models/operations"
	"openapi"
	"openapi/models/components"
	"openapi/retry"
)

func main() {
	s := openapi.New(
		openapi.WithSecurity(components.Security{
			ClientID:     "",
			ClientSecret: "",
		}),
	)

	ctx := context.Background()
	res, err := s.Wallets.V1.GetServerInfo(ctx, operations.WithRetries(
		retry.Config{
			Strategy: "backoff",
			Backoff: &retry.BackoffStrategy{
				InitialInterval: 1,
				MaxInterval:     50,
				Exponent:        1.1,
				MaxElapsedTime:  100,
			},
			RetryConnectionErrors: false,
		}))
	if err != nil {
		log.Fatal(err)
	}
	if res.ServerInfo != nil {
		// handle response
	}
}

```

If you'd like to override the default retry strategy for all operations that support retries, you can use the `WithRetryConfig` option at SDK initialization:
```go
package main

import (
	"context"
	"log"
	"openapi"
	"openapi/models/components"
	"openapi/retry"
)

func main() {
	s := openapi.New(
		openapi.WithRetryConfig(
			retry.Config{
				Strategy: "backoff",
				Backoff: &retry.BackoffStrategy{
					InitialInterval: 1,
					MaxInterval:     50,
					Exponent:        1.1,
					MaxElapsedTime:  100,
				},
				RetryConnectionErrors: false,
			}),
		openapi.WithSecurity(components.Security{
			ClientID:     "",
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
<!-- End Retries [retries] -->

<!-- Start Error Handling [errors] -->
## Error Handling

Handling errors in this SDK should largely match your expectations.  All operations return a response object or an error, they will never return both.  When specified by the OpenAPI spec document, the SDK will return the appropriate subclass.

| Error Object       | Status Code        | Content Type       |
| ------------------ | ------------------ | ------------------ |
| sdkerrors.SDKError | 4xx-5xx            | */*                |

### Example

```go
package main

import (
	"context"
	"errors"
	"log"
	"openapi"
	"openapi/models/components"
	"openapi/models/sdkerrors"
)

func main() {
	s := openapi.New(
		openapi.WithSecurity(components.Security{
			ClientID:     "",
			ClientSecret: "",
		}),
	)

	ctx := context.Background()
	res, err := s.Wallets.V1.GetServerInfo(ctx)
	if err != nil {

		var e *sdkerrors.SDKError
		if errors.As(err, &e) {
			// handle error
			log.Fatal(e.Error())
		}
	}
}

```
<!-- End Error Handling [errors] -->

<!-- Start Server Selection [server] -->
## Server Selection

### Select Server by Index

You can override the default server globally using the `WithServerIndex` option when initializing the SDK client instance. The selected server will then be used as the default on the operations that use it. This table lists the indexes associated with the available servers:

| # | Server | Variables |
| - | ------ | --------- |
| 0 | `http://localhost:8080/` | None |

#### Example

```go
package main

import (
	"context"
	"log"
	"openapi"
	"openapi/models/components"
)

func main() {
	s := openapi.New(
		openapi.WithServerIndex(0),
		openapi.WithSecurity(components.Security{
			ClientID:     "",
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


### Override Server URL Per-Client

The default server can also be overridden globally using the `WithServerURL` option when initializing the SDK client instance. For example:
```go
package main

import (
	"context"
	"log"
	"openapi"
	"openapi/models/components"
)

func main() {
	s := openapi.New(
		openapi.WithServerURL("http://localhost:8080/"),
		openapi.WithSecurity(components.Security{
			ClientID:     "",
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
<!-- End Server Selection [server] -->

<!-- Start Custom HTTP Client [http-client] -->
## Custom HTTP Client

The Go SDK makes API calls that wrap an internal HTTP client. The requirements for the HTTP client are very simple. It must match this interface:

```go
type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}
```

The built-in `net/http` client satisfies this interface and a default client based on the built-in is provided by default. To replace this default with a client of your own, you can implement this interface yourself or provide your own client configured as desired. Here's a simple example, which adds a client with a 30 second timeout.

```go
import (
	"net/http"
	"time"
	"github.com/myorg/your-go-sdk"
)

var (
	httpClient = &http.Client{Timeout: 30 * time.Second}
	sdkClient  = sdk.New(sdk.WithClient(httpClient))
)
```

This can be a convenient way to configure timeouts, cookies, proxies, custom headers, and other low-level configuration.
<!-- End Custom HTTP Client [http-client] -->

<!-- Start Authentication [security] -->
## Authentication

### Per-Client Security Schemes

This SDK supports the following security schemes globally:

| Name           | Type           | Scheme         |
| -------------- | -------------- | -------------- |
| `ClientID`     | oauth2         | OAuth2 token   |
| `ClientSecret` | oauth2         | OAuth2 token   |

You can set the security parameters through the `WithSecurity` option when initializing the SDK client instance. The selected scheme will be used by default to authenticate with the API for all operations that support it. For example:
```go
package main

import (
	"context"
	"log"
	"openapi"
	"openapi/models/components"
)

func main() {
	s := openapi.New(
		openapi.WithSecurity(components.Security{
			ClientID:     "",
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
<!-- End Authentication [security] -->

<!-- Start Special Types [types] -->
## Special Types


<!-- End Special Types [types] -->

<!-- Placeholder for Future Speakeasy SDK Sections -->

# Development

## Maturity

This SDK is in beta, and there may be breaking changes between versions without a major version update. Therefore, we recommend pinning usage
to a specific package version. This way, you can install the same version each time without breaking changes unless you are intentionally
looking for the latest version.

## Contributions

While we value open-source contributions to this SDK, this library is generated programmatically. Any manual changes added to internal files will be overwritten on the next generation. 
We look forward to hearing your feedback. Feel free to open a PR or an issue with a proof of concept and we'll do our best to include it in a future release. 

### SDK Created by [Speakeasy](https://www.speakeasy.com/?utm_source=<no value>&utm_campaign=go)
