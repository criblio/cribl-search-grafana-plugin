package plugin

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/grafana/grafana-plugin-sdk-go/backend"
)

// Represents a bearer token for use when making authenticated API calls
type BearerToken struct {
	Token     string // the bearer token itself, for use in the Authorization header
	ExpiresAt int64  // the time the token will expire (epoch milliseconds)
}

// Refresh an auth token via OAuth.  This is the normal way the plugin authenticates, using
// cribl.cloud, cribl-staging.cloud, cribl-gov.cloud, or cribl-gov-staging.cloud depending on the value of criblOrgBaseUrl.
// Upon success, returns an AuthToken which conveys the bearer token and an expiration time.
func RefreshTokenViaOAuth(criblOrgBaseUrl string, clientId string, clientSecret string) (*BearerToken, error) {
	var url, audience, requestEncoding string
	var wasGov bool = false
	var requestBody []byte
	if strings.HasSuffix(criblOrgBaseUrl, "cribl-staging.cloud") {
		url = "https://login.cribl-staging.cloud/oauth/token"
		audience = "https://api.cribl-staging.cloud"
	} else if strings.HasSuffix(criblOrgBaseUrl, "cribl-gov-staging.cloud") {
		wasGov = true
		url = "https://criblgov-stg.okta.com/oauth2/default/v1/token"
		audience = "https://api.cribl-gov-staging.cloud"
	} else if strings.HasSuffix(criblOrgBaseUrl, "cribl-gov.cloud") {
		wasGov = true
		url = "https://criblgov-prod.okta.com/oauth2/default/v1/token"
		audience = "https://api.cribl-gov.cloud"
	} else {
		url = "https://login.cribl.cloud/oauth/token"
		audience = "https://api.cribl.cloud"
	}
	backend.Logger.Debug("Refreshing token via OAuth", "url", url, "audience", audience)

	if wasGov == false {
		requestBody, _ = json.Marshal(map[string]string{
			"grant_type":    "client_credentials",
			"client_id":     clientId,
			"client_secret": clientSecret,
				"audience":      audience,
		})
		requestEncoding = "application/json"
	} else {
		requestBody = []byte(fmt.Sprintf("grant_type=client_credentials&client_id=%s&client_secret=%s&audience=%s", clientId, clientSecret, audience))
		requestEncoding = "application/x-www-form-urlencoded"
	}
	res, err := http.Post(url, requestEncoding, bytes.NewBuffer(requestBody))
	if err != nil {
		return &BearerToken{}, fmt.Errorf("auth http error: %v", err.Error())
	}
	defer res.Body.Close()
	responseBody, err := io.ReadAll(res.Body)
	if err != nil {
		return &BearerToken{}, fmt.Errorf("auth error, reading body: %v", err.Error())
	}
	if res.StatusCode != http.StatusOK {
		return &BearerToken{}, fmt.Errorf("auth error, status=%v, body=%v", res.StatusCode, string(responseBody[:]))
	}

	var oauthResponse struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int64  `json:"expires_in"`
	}
	if err = json.Unmarshal(responseBody, &oauthResponse); err != nil {
		return &BearerToken{}, fmt.Errorf("auth error, decoding body: %v", err.Error())
	}
	return &BearerToken{
		Token:     oauthResponse.AccessToken,
		ExpiresAt: time.Now().UnixMilli() + (oauthResponse.ExpiresIn * 1000),
	}, nil
}

// Refresh an auth token using the local API.  This is for local development and testing only.
// This hits the login API relative to apiBaseUrl, using the supplied username and password.
// Upon success, returns an AuthToken which conveys the bearer token and an expiration time.
func RefreshTokenViaLocalAPI(apiBaseUrl string, username string, password string) (*BearerToken, error) {
	loginUrl := fmt.Sprintf("%s/api/v1/auth/login", apiBaseUrl)
	backend.Logger.Debug("Refreshing token via local API", "url", loginUrl)

	requestBody, _ := json.Marshal(map[string]string{"username": username, "password": password})
	res, err := http.Post(loginUrl, "application/json", bytes.NewBuffer(requestBody))
	if err != nil {
		return &BearerToken{}, fmt.Errorf("login http error: %v", err.Error())
	}
	defer res.Body.Close()
	responseBody, err := io.ReadAll(res.Body)
	if err != nil {
		return &BearerToken{}, fmt.Errorf("login error, reading body: %v", err.Error())
	}
	if res.StatusCode != http.StatusOK {
		return &BearerToken{}, fmt.Errorf("login error, status=%v, body=%v", res.StatusCode, string(responseBody[:]))
	}

	var localLoginResponse struct {
		Token string `json:"token"`
	}
	if err = json.Unmarshal(responseBody, &localLoginResponse); err != nil {
		return &BearerToken{}, fmt.Errorf("login error, decoding body: %v", err.Error())
	}

	exp, err := parseExpFromJWT(localLoginResponse.Token)
	if err != nil {
		return &BearerToken{}, fmt.Errorf("login error, failed to parse JWT: %v", err.Error())
	}
	return &BearerToken{
		Token:     localLoginResponse.Token,
		ExpiresAt: exp,
	}, nil
}

// Parse the "exp" (expiration time) from a JWT and return it as epoch milliseconds
func parseExpFromJWT(jwtString string) (int64, error) {
	token, _, err := new(jwt.Parser).ParseUnverified(jwtString, jwt.MapClaims{})
	if err != nil {
		return -1, fmt.Errorf("failed to parse JWT: %v", err.Error())
	}
	exp, err := token.Claims.GetExpirationTime()
	if err != nil {
		return -1, fmt.Errorf("failed to get expiration time from JWT: %v", err.Error())
	}
	return exp.UnixMilli(), nil
}
