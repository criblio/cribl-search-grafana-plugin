package plugin

import (
	"errors"
	"fmt"
	"math"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/criblcloud/search-datasource/pkg/models"
)

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

// Convert the value of the "_time" field to an ISO timestamp string
func timeToIsoString(timeValue interface{}) (bool, string) {
	var seconds float64
	switch v := timeValue.(type) {
	case float64:
		seconds = v
	case string:
		s, err := strconv.ParseFloat(v, 64)
		if err != nil {
			return false, ""
		}
		seconds = s
	default:
		return false, ""
	}
	wholeSec := int64(seconds)
	nanoSec := int64(math.Round((seconds-float64(wholeSec))*1000.0)) * 1000000 // ms precision
	return true, time.Unix(wholeSec, nanoSec).UTC().Format(time.RFC3339Nano)
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
	default:
		return nil, fmt.Errorf("unsupported type: %T (%v)", t, t)
	}
}
