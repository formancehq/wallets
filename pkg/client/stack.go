package client

import (
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"

	sdk "github.com/formancehq/formance-sdk-go"
)

func GetStackToken() (string, error) {
	id := os.Getenv("STACK_CLIENT_ID")
	secret := os.Getenv("STACK_CLIENT_SECRET")
	endpoint := os.Getenv("STACK_URL")

	if id == "" || secret == "" {
		return "", errors.New("STACK_CLIENT_ID and STACK_CLIENT_SECRET must be set")
	}

	res, err := http.PostForm(endpoint+"/api/auth/oauth/token", url.Values{
		"grant_type":    {"client_credentials"},
		"client_id":     {id},
		"client_secret": {secret},
	})

	if err != nil {
		return "", err
	}

	defer res.Body.Close()

	if res.StatusCode != 200 {
		body, _ := io.ReadAll(res.Body)
		log.Println(string(body))
		return "", errors.New("failed to get token")
	}

	var token struct {
		AccessToken string `json:"access_token"`
	}

	if err := json.NewDecoder(res.Body).Decode(&token); err != nil {
		return "", err
	}

	return token.AccessToken, nil
}

func NewStackClient() (*sdk.APIClient, error) {
	config := sdk.NewConfiguration()
	config.Servers = sdk.ServerConfigurations{{
		URL: os.Getenv("STACK_URL"),
	}}
	// @todo: replace this with a proper token fetching and refreshing mechanism
	token, err := GetStackToken()
	if err != nil {
		return nil, err
	}
	config.AddDefaultHeader("Authorization", "Bearer "+token)

	return sdk.NewAPIClient(config), nil
}
