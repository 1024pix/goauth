package credentials

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/grokify/oauth2more"
	"github.com/grokify/oauth2more/endpoints"
	"github.com/grokify/simplego/net/http/httpsimple"
	"github.com/pkg/errors"
	"golang.org/x/oauth2"
)

const (
	TypeOAuth2 = "oauth2"
	TypeJWT    = "jwt"
)

type Credentials struct {
	Service             string                 `json:"service,omitempty"`
	Type                string                 `json:"type,omitempty"`
	Subdomain           string                 `json:"subdomain,omitempty"`
	Application         ApplicationCredentials `json:"application,omitempty"`
	PasswordCredentials PasswordCredentials    `json:"passwordCredentials,omitempty"`
	JWT                 JWTCredentials         `json:"jwt,omitempty"`
	Token               *oauth2.Token          `json:"token,omitempty"`
}

func NewCredentialsJSON(data []byte) (Credentials, error) {
	var creds Credentials
	err := json.Unmarshal(data, &creds)
	if err != nil {
		return creds, err
	}
	err = creds.Inflate()
	return creds, err
}

func NewCredentialsJSONs(appJson, userJson, accessToken []byte) (Credentials, error) {
	var creds Credentials
	if len(appJson) > 1 {
		app := ApplicationCredentials{}
		err := json.Unmarshal(appJson, &app)
		if err != nil {
			return creds, err
		}
		creds.Application = app
	}
	if len(userJson) > 0 {
		user := PasswordCredentials{}
		err := json.Unmarshal(userJson, &user)
		if err != nil {
			return creds, err
		}
		creds.PasswordCredentials = user
	}
	if len(accessToken) > 0 {
		creds.Token = &oauth2.Token{
			AccessToken: string(accessToken)}
	}
	return creds, nil
}

func (creds *Credentials) Inflate() error {
	if len(strings.TrimSpace(creds.Service)) > 0 {
		ep, svcUrl, err := endpoints.NewEndpoint(creds.Service, creds.Subdomain)
		if err != nil {
			return err
		}
		if creds.Application.OAuth2Endpoint == (oauth2.Endpoint{}) {
			creds.Application.OAuth2Endpoint = ep
		}
		if len(strings.TrimSpace(creds.Application.ServerURL)) == 0 {
			creds.Application.ServerURL = svcUrl
		}
	}
	return nil
}

func (creds *Credentials) NewClient() (*http.Client, error) {
	tok := creds.Token
	if tok == nil {
		var err error
		tok, err = creds.NewToken()
		if err != nil {
			return nil, errors.Wrap(err, "Credentials.NewClient()")
		}
		creds.Token = tok
	}
	return oauth2more.NewClientToken(
		oauth2more.TokenBearer, tok.AccessToken, false), nil
}

func (creds *Credentials) NewSimpleClient(httpClient *http.Client) (*httpsimple.SimpleClient, error) {
	return &httpsimple.SimpleClient{
		BaseURL:    creds.Application.ServerURL,
		HTTPClient: httpClient}, nil
}

func (creds *Credentials) NewClientCli(oauth2State string) (*http.Client, error) {
	tok, err := creds.NewTokenCli(oauth2State)
	if err != nil {
		return nil, err
	}
	creds.Token = tok
	return oauth2more.NewClientToken(
		oauth2more.TokenBearer, tok.AccessToken, false), nil
}

func (creds *Credentials) NewToken() (*oauth2.Token, error) {
	cfg := creds.Application.Config()
	return cfg.PasswordCredentialsToken(
		context.Background(),
		creds.PasswordCredentials.Username,
		creds.PasswordCredentials.Password)
}

// NewTokenCli retrieves a token using CLI approach for
// OAuth 2.0 authorization code or password grant.
func (creds *Credentials) NewTokenCli(oauth2State string) (*oauth2.Token, error) {
	if strings.ToLower(strings.TrimSpace(creds.Application.GrantType)) == "code" {
		return NewTokenCli(*creds, oauth2State)
	}
	return creds.NewToken()
}
