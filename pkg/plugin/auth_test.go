package plugin

import (
	"crypto/tls"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"
)

func TestLocalAuth(t *testing.T) {
	apiBaseUrl, username, password := os.Getenv("LOCAL_API_BASE_URL"), os.Getenv("LOCAL_API_USERNAME"), os.Getenv("LOCAL_API_PASSWORD")
	if apiBaseUrl == "" {
		t.Skip("LOCAL_API_BASE_URL not defined, skipping local auth test")
	}
	if username == "" {
		username = "admin"
	}
	if password == "" {
		password = "admin"
	}
	// Use HTTP client with InsecureSkipVerify for local testing
	httpClient := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	}
	token, err := RefreshTokenViaLocalAPI(apiBaseUrl, username, password, httpClient)
	if err != nil {
		t.Fatalf("failed to refresh local auth token: %v", err.Error())
	}
	validateToken(token, t)
}

func TestCloudAuth(t *testing.T) {
	criblOrgBaseUrl, clientId, clientSecret := os.Getenv("CRIBL_ORG_BASE_URL"), os.Getenv("CLIENT_ID"), os.Getenv("CLIENT_SECRET")
	if criblOrgBaseUrl == "" || clientId == "" || clientSecret == "" {
		t.Skip("CRIBL_ORG_BASE_URL / CLIENT_ID / CLIENT_SECRET not defined, skipping cloud auth test")
	}
	// Use default HTTP client for cloud auth
	httpClient := http.DefaultClient
	token, err := RefreshTokenViaOAuth(criblOrgBaseUrl, clientId, clientSecret, httpClient)
	if err != nil {
		t.Fatalf("failed to refresh local auth token: %v", err.Error())
	}
	validateToken(token, t)
}

func validateToken(token *BearerToken, t *testing.T) {
	if len(token.Token) <= 0 {
		t.Fatal("Empty token")
	}
	if len(strings.Split(token.Token, ".")) != 3 {
		t.Fatal("Invalid token, expected 3 dot-delimited components")
	}
	if token.ExpiresAt <= time.Now().UnixMilli() {
		t.Fatal("Invalid token, ExpiresAt is in the past")
	}
}
