package oauth2util

import (
	"encoding/base64"
	"fmt"
	"strings"

	"github.com/grokify/gotilla/time/timeutil"
	"golang.org/x/oauth2"
)

// RFC7617UserPass base64 encodes a user-id and password per:
// https://tools.ietf.org/html/rfc7617#section-2
func RFC7617UserPass(userid, password string) (string, error) {
	if strings.Index(userid, ":") > -1 {
		return "", fmt.Errorf(
			"RFC7617 user-id cannot include a colon (':') [%v]", userid)
	}
	userpass := strings.Join([]string{userid, password}, ":")
	return base64.StdEncoding.EncodeToString([]byte(userpass)), nil
}

func BasicAuthToken(username, password string) (*oauth2.Token, error) {
	basicToken, err := RFC7617UserPass(username, password)
	if err != nil {
		return nil, err
	}

	return &oauth2.Token{
		AccessToken: basicToken,
		TokenType:   "Basic",
		Expiry:      timeutil.TimeRFC3339Zero()}, nil
}
