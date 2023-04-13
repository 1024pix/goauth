package goauth

import (
	"fmt"
	"strings"
	"time"

	"github.com/grokify/goauth/authutil"
	"golang.org/x/oauth2"
)

func NewTokenCLI(creds Credentials, state string) (token *oauth2.Token, err error) {
	if creds.OAuth2.IsGrantType(authutil.GrantTypeAuthorizationCode) {
		state = strings.TrimSpace(state)
		if len(state) == 0 {
			state = "goauth-" + time.Now().UTC().Format(time.RFC3339)
		}
		fmt.Printf("OAuth State [%s]\n", state)
		cfg := creds.OAuth2.Config()
		token, err = authutil.NewTokenCLIFromWeb(&cfg, state)
		if err != nil {
			return token, err
		}
	} else {
		token, err = creds.NewToken()
		if err != nil {
			return token, err
		}
	}
	token.Expiry = token.Expiry.UTC()
	return token, nil
}
