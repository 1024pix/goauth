package ringcentral

import (
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	ju "github.com/grokify/gotilla/encoding/jsonutil"
	hum "github.com/grokify/gotilla/net/httputilmore"
	om "github.com/grokify/oauth2more"
	"golang.org/x/oauth2"
)

var (
	EnvServerURL    = "RINGCENTRAL_SERVER_URL"
	EnvClientID     = "RINGCENTRAL_CLIENT_ID"
	EnvClientSecret = "RINGCENTRAL_CLIENT_SECRET"
	EnvAppName      = "RINGCENTRAL_APP_NAME"
	EnvAppVersion   = "RINGCENTRAL_APP_VERSION"
	EnvRedirectURL  = "RINGCENTRAL_OAUTH_REDIRECT_URL"
	EnvUsername     = "RINGCENTRAL_USERNAME"
	EnvExtension    = "RINGCENTRAL_EXTENSION"
	EnvPassword     = "RINGCENTRAL_PASSWORD"
)

type ApplicationCredentials struct {
	ServerURL       string
	ApplicationID   string
	ClientID        string
	ClientSecret    string
	RedirectURL     string
	AppName         string
	AppVersion      string
	OAuthEndpointID string
	AccessTokenTTL  int64
	RefreshTokenTTL int64
}

func (ac *ApplicationCredentials) AppNameAndVersion() string {
	parts := []string{}
	ac.AppName = strings.TrimSpace(ac.AppName)
	ac.AppVersion = strings.TrimSpace(ac.AppVersion)
	if len(ac.AppName) > 0 {
		parts = append(parts, ac.AppName)
	}
	if len(ac.AppVersion) > 0 {
		parts = append(parts, fmt.Sprintf("v%v", ac.AppVersion))
	}
	return strings.Join(parts, "-")
}

func (app *ApplicationCredentials) Config() oauth2.Config {
	return oauth2.Config{
		ClientID:     app.ClientID,
		ClientSecret: app.ClientSecret,
		Endpoint:     NewEndpoint(app.ServerURL),
		RedirectURL:  app.RedirectURL}
}

func (app *ApplicationCredentials) Exchange(code string) (*RcToken, error) {
	params := url.Values{}
	params.Set("grant_type", "authorization_code")
	params.Set("code", code)
	params.Set("redirect_uri", app.RedirectURL)
	if len(app.OAuthEndpointID) > 0 {
		params.Set("endpoint_id", app.OAuthEndpointID)
	}
	if app.AccessTokenTTL > 0 {
		params.Set("accessTokenTtl", strconv.Itoa(int(app.AccessTokenTTL)))
	}
	if app.RefreshTokenTTL > 0 {
		params.Set("refreshTokenTtl", strconv.Itoa(int(app.RefreshTokenTTL)))
	}
	return RetrieveRcToken(app.Config(), params)
}

type UserCredentials struct {
	Username  string
	Extension string
	Password  string
}

func (uc *UserCredentials) UsernameSimple() string {
	if len(strings.TrimSpace(uc.Extension)) > 0 {
		return strings.Join([]string{uc.Username, uc.Extension}, "*")
	}
	return uc.Username
}

func NewTokenPassword(app ApplicationCredentials, pwd PasswordCredentials) (*oauth2.Token, error) {
	return RetrieveToken(
		oauth2.Config{
			ClientID:     app.ClientID,
			ClientSecret: app.ClientSecret,
			Endpoint:     NewEndpoint(app.ServerURL)},
		pwd.URLValues())
}

// NewClientPassword uses dedicated password grant handling.
func NewClientPassword(app ApplicationCredentials, pwd PasswordCredentials) (*http.Client, error) {
	c := app.Config()
	token, err := RetrieveToken(c, pwd.URLValues())
	if err != nil {
		return nil, err
	}

	httpClient := c.Client(oauth2.NoContext, token)

	header := getClientHeader(app)
	if len(header) > 0 {
		httpClient.Transport = hum.TransportWithHeaders{
			Transport: httpClient.Transport,
			Header:    header}
	}
	return httpClient, nil
}

// NewClientPasswordSimple uses OAuth2 package password grant handling.
func NewClientPasswordSimple(app ApplicationCredentials, user UserCredentials) (*http.Client, error) {
	httpClient, err := om.NewClientPasswordConf(
		oauth2.Config{
			ClientID:     app.ClientID,
			ClientSecret: app.ClientSecret,
			Endpoint:     NewEndpoint(app.ServerURL)},
		user.UsernameSimple(),
		user.Password)
	if err != nil {
		return nil, err
	}

	header := getClientHeader(app)
	if len(header) > 0 {
		httpClient.Transport = hum.TransportWithHeaders{
			Transport: httpClient.Transport,
			Header:    header}
	}
	return httpClient, nil
}

func getClientHeader(app ApplicationCredentials) http.Header {
	userAgentParts := []string{om.PathVersion()}
	if len(app.AppNameAndVersion()) > 0 {
		userAgentParts = append([]string{app.AppNameAndVersion()}, userAgentParts...)
	}
	userAgent := strings.TrimSpace(strings.Join(userAgentParts, "; "))

	header := http.Header{}
	if len(userAgent) > 0 {
		header.Add(hum.HeaderUserAgent, userAgent)
		header.Add("X-User-Agent", userAgent)
	}
	return header
}

func NewClientPasswordEnv() (*http.Client, error) {
	return NewClientPassword(
		NewApplicationCredentialsEnv(),
		NewPasswordCredentialsEnv())
}

func NewApplicationCredentialsEnv() ApplicationCredentials {
	return ApplicationCredentials{
		ServerURL:    os.Getenv(EnvServerURL),
		ClientID:     os.Getenv(EnvClientID),
		ClientSecret: os.Getenv(EnvClientSecret),
		AppName:      os.Getenv(EnvAppName),
		AppVersion:   os.Getenv(EnvAppVersion)}
}

func NewPasswordCredentialsEnv() PasswordCredentials {
	return PasswordCredentials{
		Username:  os.Getenv(EnvUsername),
		Extension: os.Getenv(EnvExtension),
		Password:  os.Getenv(EnvPassword)}
}

type PasswordCredentials struct {
	GrantType       string `url:"grant_type"`
	AccessTokenTTL  int64  `url:"access_token_ttl"`
	RefreshTokenTTL int64  `url:"refresh_token_ttl"`
	Username        string `url:"username"`
	Extension       string `url:"extension"`
	Password        string `url:"password"`
	EndpointId      string `url:"endpoint_id"`
}

func (pw *PasswordCredentials) URLValues() url.Values {
	v := url.Values{
		"grant_type": {"password"},
		"username":   {pw.Username},
		"password":   {pw.Password}}
	if pw.AccessTokenTTL != 0 {
		v.Set("access_token_ttl", strconv.Itoa(int(pw.AccessTokenTTL)))
	}
	if pw.RefreshTokenTTL != 0 {
		v.Set("refresh_token_ttl", strconv.Itoa(int(pw.RefreshTokenTTL)))
	}
	if len(pw.Extension) > 0 {
		v.Set("extension", pw.Extension)
	}
	if len(pw.EndpointId) > 0 {
		v.Set("endpoint_id", pw.EndpointId)
	}
	return v
}

func RetrieveToken(cfg oauth2.Config, params url.Values) (*oauth2.Token, error) {
	rcToken, err := RetrieveRcToken(cfg, params)
	if err != nil {
		return nil, err
	}
	return rcToken.OAuth2Token()
}

func RetrieveRcToken(cfg oauth2.Config, params url.Values) (*RcToken, error) {
	r, err := http.NewRequest(
		http.MethodPost,
		cfg.Endpoint.TokenURL,
		strings.NewReader(params.Encode()))
	if err != nil {
		return nil, err
	}

	basicAuthHeader, err := om.BasicAuthHeader(cfg.ClientID, cfg.ClientSecret)
	if err != nil {
		return nil, err
	}

	r.Header.Add(hum.HeaderAuthorization, basicAuthHeader)
	r.Header.Add(hum.HeaderContentType, hum.ContentTypeAppFormUrlEncoded)
	r.Header.Add(hum.HeaderContentLength, strconv.Itoa(len(params.Encode())))

	client := &http.Client{}
	resp, err := client.Do(r)
	if err != nil {
		return nil, err
	} else if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("RingCentral API Response Status [%v]", resp.StatusCode)
	}

	rcToken := &RcToken{}
	_, err = ju.UnmarshalIoReader(resp.Body, rcToken)
	//err = hum.UnmarshalResponseJSON(resp, rcToken)
	if err != nil {
		return nil, err
	}
	err = rcToken.Inflate()
	return rcToken, err
}

type RcToken struct {
	AccessToken           string    `json:"access_token,omitempty"`
	TokenType             string    `json:"token_type,omitempty"`
	Scope                 string    `json:"scope,omitempty"`
	ExpiresIn             int64     `json:"expires_in,omitempty"`
	RefreshToken          string    `json:"refresh_token,omitempty"`
	RefreshTokenExpiresIn int64     `json:"refresh_token_expires_in,omitempty"`
	OwnerID               string    `json:"owner_id,omitempty"`
	EndpointID            string    `json:"endpoint_id,omitempty"`
	Expiry                time.Time `json:"expiry,omitempty"`
	RefreshTokenExpiry    time.Time `json:"refresh_token_expiry,omitempty"`
	inflated              bool      `json:"inflated"`
}

func (rcTok *RcToken) Inflate() error {
	now := time.Now()
	if (rcTok.ExpiresIn) > 0 {
		expiresIn, err := time.ParseDuration(fmt.Sprintf("%vs", rcTok.ExpiresIn))
		if err != nil {
			return err
		}
		rcTok.Expiry = now.Add(expiresIn)
	}
	if (rcTok.RefreshTokenExpiresIn) > 0 {
		expiresIn, err := time.ParseDuration(fmt.Sprintf("%vs", rcTok.RefreshTokenExpiresIn))
		if err != nil {
			return err
		}
		rcTok.RefreshTokenExpiry = now.Add(expiresIn)
	}

	rcTok.inflated = true
	return nil
}

func (rcTok *RcToken) OAuth2Token() (*oauth2.Token, error) {
	if !rcTok.inflated {
		err := rcTok.Inflate()
		if err != nil {
			return nil, err
		}
	}

	tok := &oauth2.Token{
		AccessToken:  rcTok.AccessToken,
		TokenType:    rcTok.TokenType,
		RefreshToken: rcTok.RefreshToken,
		Expiry:       rcTok.Expiry}

	return tok, nil
}
