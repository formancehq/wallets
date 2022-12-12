package client

import (
	"os"

	sdk "github.com/formancehq/formance-sdk-go"
)

func NewStackClient() (*sdk.APIClient, error) {
	config := sdk.NewConfiguration()
	config.Servers = sdk.ServerConfigurations{{
		URL: os.Getenv("STACK_URL"),
	}}
	config.AddDefaultHeader("Authorization", "Bearer "+os.Getenv("STACK_TOKEN"))

	return sdk.NewAPIClient(config), nil
}
