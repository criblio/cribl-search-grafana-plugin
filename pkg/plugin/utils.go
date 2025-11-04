package plugin

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net/url"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/criblcloud/search-datasource/pkg/models"
)

// pluginVersion holds the version of the plugin, set at startup
var pluginVersion = "dev"

// SetVersion sets the plugin version (called from main)
func SetVersion(version string) {
	pluginVersion = version
}

// i.e. for use in composing the User-Agent header
func GetVersion() string {
	return pluginVersion
}

// Can the supplied query be run as-is?
func canRunQuery(criblQuery *models.CriblQuery) error {
	switch criblQuery.Type {
	case "adhoc":
		if len(strings.TrimSpace(criblQuery.Query)) == 0 {
			return errors.New("query is empty")
		}
	case "saved":
		if len(criblQuery.SavedSearchId) == 0 {
			return errors.New("saved search ID is missing")
		}
	default:
		return fmt.Errorf("unsupported query type: %v", criblQuery.Type)
	}
	return nil
}

// Cribl Search backend requires a fully-formed query including the "cribl" operator.
// Here we're applying best effort to auto-prepending the "cribl" operator where it needs to be.
// This is far from 100% foolproof, but covers 99% of the happy paths.
func prependCriblOperator(query string) string {
	// This tries to capture the first word of the root query statement, along with any preceding
	// "set" or "let" statements, and everything that comes after the first word.  NOTE: This doesn't
	// touch any "let" statements.  We might enhance that in the future, but for now, users need to
	// put their own "cribl" operator in those stage statements.
	re := regexp.MustCompile(`^((?:\s*(?:set|let)\s+[^;]+;)*\s*)((\w+|['"*]).*)$`)
	match := re.FindStringSubmatch(strings.TrimSpace(query))
	if match != nil {
		firstWord := match[3]
		// We recognize certain operators that don't need "cribl" prepended
		if !slices.Contains([]string{"cribl", "externaldata", "find", "print", "search", "range"}, firstWord) {
			return fmt.Sprintf("%scribl %s", match[1], match[2])
		}
	}
	return query
}

// Easier to troubleshoot a query from the logs when it's a single line, space instead of tab, etc.
func collapseToSingleLine(query string) string {
	return regexp.MustCompile("[\r\n\t]+").ReplaceAllString(query, " ")
}

// Convert the value of the "_time" field (expected to be in seconds) to a time.Time in UTC
func criblTimeToGrafanaTime(timeValue interface{}) (bool, time.Time) {
	var seconds float64
	switch v := timeValue.(type) {
	case float64:
		seconds = v
	case string:
		s, err := strconv.ParseFloat(v, 64)
		if err != nil {
			return false, time.Time{}
		}
		seconds = s
	default:
		return false, time.Time{}
	}
	wholeSec := int64(seconds)
	nanoSec := int64(math.Round((seconds-float64(wholeSec))*1000000.0)) * 1000 // microsec precision
	return true, time.Unix(wholeSec, nanoSec).UTC()
}

// Grafana's data.NewField() is super finicky.  You're force to supply an array of values,
// and that array must have a concrete type.  Unfortunately, we've unmarshalled results from JSON
// and values are `interface{}`, and Grafana doesn't allow arbitrary values like that.  So here
// we're determining the basic type and constructing an empty array of that concrete type so we
// can pass it to data.NewField().
func makeEmptyConcreteTypeArray(val interface{}) (interface{}, error) {
	switch t := val.(type) {
	case string:
		return []string{}, nil
	case float64:
		return []float64{}, nil
	case bool:
		return []bool{}, nil
	case time.Time:
		return []time.Time{}, nil
	default:
		return nil, fmt.Errorf("unsupported type: %T (%v)", t, t)
	}
}

// Grafana doesn't like nested values.  If a field value is an object, flatten it
// to a string by serializing it to JSON.
func flattenNestedObjectToString(val interface{}) interface{} {
	switch val.(type) {
	case map[string]interface{}:
		if b, err := json.Marshal(val); err == nil {
			return string(b)
		}
	}
	return val
}

// Parse a serialized JavaScript error.  Expects at least `name` and `message` fields, and possibly
// extra fields.  Returns an error if it was successfully parsed, otherwise nil.
func parseJavaScriptError(body []byte) error {
	var fields map[string]interface{}
	if err := json.Unmarshal(body, &fields); err != nil {
		return nil // not JSON
	}
	if fields["name"] == nil || fields["message"] == nil {
		return nil // not a JavaScript error
	}

	name := fields["name"].(string)
	message := fields["message"].(string)

	// See if there are any other fields on the error, i.e. "code" or what not
	delete(fields, "name")
	delete(fields, "message")
	if len(fields) == 0 {
		return fmt.Errorf("%v: %v", name, message) // nope, no extra fields
	}

	// Include the extra fields
	var extras []string
	for key, value := range fields {
		extras = append(extras, fmt.Sprintf("%s: %v", key, value))
	}
	return fmt.Errorf("%v: %v (%+v)", name, message, strings.Join(extras, ", "))
}

// Determine if a supplied URL is well-formed, i.e. including scheme and host
func isValidURL(rawURL string) bool {
	u, err := url.Parse(rawURL)
	return err == nil && (u.Scheme == "https" || u.Scheme == "http") && u.Host != ""
}

// Check if a URL is for local development (localhost or Docker internal networking)
// Returns true if the hostname is "localhost" or "host.docker.internal"
func isLocalDevelopmentURL(rawURL string) bool {
	u, err := url.Parse(rawURL)
	if err != nil {
		return false
	}
	hostname := u.Hostname()
	return hostname == "localhost" || hostname == "host.docker.internal"
}
