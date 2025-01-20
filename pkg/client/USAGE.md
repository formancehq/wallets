<!-- Start SDK Example Usage [usage] -->
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