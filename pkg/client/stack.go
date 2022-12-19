package client

import (
	"context"
	"net/http"
	"os"

	sdk "github.com/formancehq/formance-sdk-go"
	"github.com/pkg/errors"
	"golang.org/x/oauth2/clientcredentials"
)

func GetAuthenticatedClient(ctx context.Context) (*http.Client, error) {
	id := os.Getenv("STACK_CLIENT_ID")
	secret := os.Getenv("STACK_CLIENT_SECRET")
	endpoint := os.Getenv("STACK_URL")

	if id == "" || secret == "" {
		return nil, errors.New("STACK_CLIENT_ID and STACK_CLIENT_SECRET must be set")
	}

	clientCredentialsConfig := clientcredentials.Config{
		ClientID:     id,
		ClientSecret: secret,
		TokenURL:     endpoint + "/api/auth/oauth/token",
	}
	return clientCredentialsConfig.Client(ctx), nil
}

func NewStackClient() (*sdk.APIClient, error) {
	config := sdk.NewConfiguration()
	config.Servers = sdk.ServerConfigurations{{
		URL: os.Getenv("STACK_URL"),
	}}

	httpClient, err := GetAuthenticatedClient(context.Background())
	if err != nil {
		return nil, err
	}
	config.HTTPClient = httpClient

	return sdk.NewAPIClient(config), nil
}
