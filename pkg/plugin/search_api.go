package plugin

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/criblcloud/search-datasource/pkg/models"
	"github.com/grafana/grafana-plugin-sdk-go/backend"
)

func NewSearchAPI(settings *models.PluginSettings) *SearchAPI {
	return &SearchAPI{
		Settings:    settings,
		BearerToken: nil,
	}
}

type SearchAPI struct {
	Settings    *models.PluginSettings
	BearerToken *BearerToken
}

type SearchQueryResult struct {
	Header map[string]interface{}
	Events []map[string]interface{}
}

// Run a search query and return the header event + result events.  The queryParams arg is
// expected to have params such as query + earlieset + latest, or a savedSearchId, and
// any offset + limit as needed.  This simply makes the API request and parses the response.
func (api *SearchAPI) RunQueryAndGetResults(queryParams *url.Values) (*SearchQueryResult, error) {
	responseBytes, err := api.doGET("/api/v1/m/default_search/search/query", queryParams)
	if err != nil {
		return nil, err
	}
	// The response is NDJSON, one header "event" plus result events
	lines := strings.Split(string(responseBytes), "\n")
	result := SearchQueryResult{}
	for idx, line := range lines {
		if len(line) == 0 {
			continue
		}
		var event map[string]interface{}
		if err := json.Unmarshal([]byte(line), &event); err != nil {
			return nil, fmt.Errorf("failed to parse json at line %d: %s", idx+1, line)
		}
		if idx == 0 {
			result.Header = event
		} else {
			result.Events = append(result.Events, event)
		}
	}
	return &result, nil
}

// Cancel a search query.
func (api *SearchAPI) CancelQuery(jobId string) error {
	_, err := api.doPOST(fmt.Sprintf("/api/v1/m/default_search/search/jobs/%s/cancel", jobId), nil, "application/json", []byte("{}"))
	return err
}

// Load the list of saved search IDs available to the user corresponding to the API creds.
// This can be used to populate a dropdown to make it easy for the user to pick one.
// Returns a list of saved search IDs.
func (api *SearchAPI) LoadSavedSearchIds() ([]string, error) {
	responseBytes, err := api.doGET("/api/v1/m/default_search/search/saved", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to load saved search ids: %v", err.Error())
	}
	var data struct {
		Items []map[string]any `json:"items"`
	}
	if err = json.Unmarshal(responseBytes, &data); err != nil {
		return nil, fmt.Errorf("failed to load saved search ids: error while parsing JSON: %v", err.Error())
	}

	var ids []string
	for _, item := range data.Items {
		ids = append(ids, item["id"].(string))
	}
	return ids, nil
}

// Perform a GET request to the API, returning the raw response body as a byte array
func (api *SearchAPI) doGET(uri string, queryParams *url.Values) ([]byte, error) {
	req, err := http.NewRequest("GET", api.url(uri), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create GET request: %v", err.Error())
	}
	err = api.addAuthorization(req)
	if err != nil {
		return nil, fmt.Errorf("failed to add Authorization: %v", err.Error())
	}
	if queryParams != nil {
		req.URL.RawQuery = queryParams.Encode()
	}
	backend.Logger.Debug("http GET", "URL", req.URL.String())
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("GET request failed: %v", err.Error())
	}
	return api.readResponse(res)
}

// Perform a GET request to the API, returning the raw response body as a byte array
func (api *SearchAPI) doPOST(uri string, queryParams *url.Values, contentType string, data []byte) ([]byte, error) {
	req, err := http.NewRequest("POST", api.url(uri), bytes.NewBuffer(data))
	if err != nil {
		return nil, fmt.Errorf("failed to create POST request: %v", err.Error())
	}
	err = api.addAuthorization(req)
	if err != nil {
		return nil, fmt.Errorf("failed to add Authorization: %v", err.Error())
	}
	if queryParams != nil {
		req.URL.RawQuery = queryParams.Encode()
	}
	req.Header.Set("Content-Type", contentType)
	backend.Logger.Debug("http POST", "URL", req.URL.String(), "contentType", contentType)
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("POST request failed: %v", err.Error())
	}
	return api.readResponse(res)
}

func (api *SearchAPI) readResponse(res *http.Response) ([]byte, error) {
	defer res.Body.Close()
	responseBody, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %v", err.Error())
	}
	if res.StatusCode != http.StatusOK {
		// Try to parse the error from the response
		if err := parseErrorFromResponse(responseBody); err != nil {
			return nil, err
		}
		// Couldn't parse the error from the response, just return a generalized error
		return nil, fmt.Errorf("request failed (%v): %v", res.StatusCode, string(responseBody))
	}
	return responseBody, nil
}

// Compose the full URL to an API resource
func (api *SearchAPI) url(path string) string {
	return fmt.Sprintf("%s%s", api.Settings.CriblOrgBaseUrl, path)
}

// Add the Authorization header to an http.Request, refreshing our cached authentication as needed
func (api *SearchAPI) addAuthorization(req *http.Request) error {
	err := api.refreshBearerTokenAsNeeded()
	if err != nil {
		return err
	}
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", api.BearerToken.Token))
	return nil
}

// Establish the cached bearer token, refreshing as needed.  This honors the expiration time
// but applies a 30-second buffer to avoid cutting it too close.  Bearer tokens are typically
// valid for many hours.
func (api *SearchAPI) refreshBearerTokenAsNeeded() error {
	if api.BearerToken != nil && api.BearerToken.ExpiresAt > (time.Now().UnixMilli()+30000) {
		backend.Logger.Debug("Reusing cached bearer token", "ExpiresAt", api.BearerToken.ExpiresAt)
		return nil // current token is still valid
	}

	backend.Logger.Debug("Refreshing bearer token")
	var err error
	if strings.HasSuffix(api.Settings.CriblOrgBaseUrl, ".cloud") { // i.e. foo.cribl.cloud or bar.cribl-staging.cloud
		api.BearerToken, err = RefreshTokenViaOAuth(api.Settings.CriblOrgBaseUrl, api.Settings.ClientId, api.Settings.Secrets.ClientSecret)
	} else {
		api.BearerToken, err = RefreshTokenViaLocalAPI(api.Settings.CriblOrgBaseUrl, api.Settings.ClientId, api.Settings.Secrets.ClientSecret)
	}
	return err
}

// Try to parse an error from an API response.  Returns nil if for any reason we couldn't
// parse the error, or if the response was an unexpected format.  We attempt to unpack the body
// to whatever extent possible.  Normally we expect a JSON object with "status" and "message"
// fields.  Often the "message" field itself is a serialized JavaScript Error object.  We make
// an attempt to provide the most user-friendly representation of the error.
func parseErrorFromResponse(body []byte) error {
	var jsonBody map[string]interface{}
	if err := json.Unmarshal(body, &jsonBody); err != nil {
		return nil // it's not JSON
	}

	if jsonBody["message"] == nil {
		return nil // it's not the format we expected
	}

	// See if there's a serialized JavaScript Error in the message itself
	if err := parseJavaScriptError([]byte(jsonBody["message"].(string))); err != nil {
		return err
	}

	return errors.New(jsonBody["message"].(string)) // not a JS Error, return the message as-is
}
