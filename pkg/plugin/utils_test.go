package plugin

import (
	"fmt"
	"testing"

	"github.com/criblcloud/search-datasource/pkg/models"
	"github.com/stretchr/testify/assert"
)

func TestCanRunQuery(t *testing.T) {
	if canRunQuery(&models.CriblQuery{}) == nil {
		t.Fatal("should not be able to run with no type")
	}

	if canRunQuery(&models.CriblQuery{Type: "adhoc"}) == nil {
		t.Fatal("should not be able to run adhoc with no query")
	}
	if canRunQuery(&models.CriblQuery{Type: "adhoc", Query: ""}) == nil {
		t.Fatal("should not be able to run adhoc with empty query")
	}
	if canRunQuery(&models.CriblQuery{Type: "adhoc", Query: "      "}) == nil {
		t.Fatal("should not be able to run adhoc with blank query")
	}

	if canRunQuery(&models.CriblQuery{Type: "saved"}) == nil {
		t.Fatal("should not be able to run saved with no savedSearchId")
	}
	if canRunQuery(&models.CriblQuery{Type: "adhoc", SavedSearchId: ""}) == nil {
		t.Fatal("should not be able to run saved with empty savedSearchId")
	}
}

func TestPrependCriblOperator(t *testing.T) {
	for _, test := range []struct {
		In       string
		Expected string
	}{
		{
			In:       `dataset="foo" | where something | timestats by level`,
			Expected: `cribl dataset="foo" | where something | timestats by level`,
		},
		{
			In:       `cribl something`,
			Expected: `cribl something`,
		},
		{
			In:       `set logger_level="debug"; dataset="foo"`,
			Expected: `set logger_level="debug"; cribl dataset="foo"`,
		},
		{
			In:       `set logger_level="debug";whatever`,
			Expected: `set logger_level="debug";cribl whatever`,
		},
	} {
		out := prependCriblOperator(test.In)
		assert.Equal(t, test.Expected, out, test.In)
	}
}

func TestCollapseToSingleLine(t *testing.T) {
	assert.Equal(t, `hello there dude`, collapseToSingleLine("hello\nthere\ndude"))
	assert.Equal(t, `hello there aw yeah`, collapseToSingleLine("hello\nthere\taw\r\nyeah"))
}

func TestCriblTimeToGrafanaTime(t *testing.T) {
	for _, test := range []struct {
		In       interface{}
		Expected int64
	}{
		{
			In:       nil,
			Expected: 0,
		},
		{
			In:       false,
			Expected: 0,
		},
		{
			In:       true,
			Expected: 0,
		},
		{
			In:       "whatever",
			Expected: 0,
		},
		{
			In:       float64(1728744793),
			Expected: 1728744793000000,
		},
		{
			In:       float64(1728744793.123),
			Expected: 1728744793123000,
		},
		{
			In:       float64(1728744793.123456),
			Expected: 1728744793123456,
		},
	} {
		ok, out := criblTimeToGrafanaTime(test.In)
		if test.Expected == 0 {
			assert.Equal(t, false, ok, fmt.Sprintf("input %v produced ok=%v, out=%v", test.In, ok, out))
		} else {
			assert.Equal(t, true, ok, fmt.Sprintf("input %v produced ok=%v, out=%v", test.In, ok, out))
			assert.Equal(t, test.Expected, out.UnixMicro(), test.In)
		}
	}
}

func TestParseJavaScriptError(t *testing.T) {
	for _, test := range []struct {
		In       string
		Expected interface{}
	}{
		{
			In:       `not even json`,
			Expected: nil,
		},
		{
			In:       `{"no":"message or name fields"}`,
			Expected: nil,
		},
		{
			In:       `{"message":"no name field here"}`,
			Expected: nil,
		},
		{
			In:       `{"name":"no message field here"}`,
			Expected: nil,
		},
		{
			In:       `{"name":"AwesomeError","message":"This error has no extra fields."}`,
			Expected: `AwesomeError: This error has no extra fields.`,
		},
		{
			In: `{"name":"AwesomeError","message":"This error does have extra fields.","code":42,"foo_bar":false}`,
			Expected: []string{
				// json.Unmarshal() produces a map whose keys are in random order...can be either of these
				`AwesomeError: This error does have extra fields. (code: 42, foo_bar: false)`,
				`AwesomeError: This error does have extra fields. (foo_bar: false, code: 42)`,
			},
		},
	} {
		err := parseJavaScriptError([]byte(test.In))
		if test.Expected == nil {
			assert.Nil(t, err, fmt.Sprintf("expected nil for %v", test.In))
		} else {
			assert.NotNil(t, err, fmt.Sprintf("expected non-nil for %v", test.In))
			if strSlice, ok := test.Expected.([]string); ok {
				anyMatched := false
				for _, str := range strSlice {
					anyMatched = anyMatched || str == err.Error()
				}
				assert.True(t, anyMatched, "unexpected output format")
			} else {
				assert.Equal(t, test.Expected.(string), err.Error(), "unexpected output format")
			}
		}
	}
}

func TestIsValidURL(t *testing.T) {
	assert.False(t, isValidURL(""), "empty string")
	assert.False(t, isValidURL(" "), "blank string")
	assert.False(t, isValidURL("something"), "no scheme")
	assert.False(t, isValidURL("foo://something"), "invalid scheme")
	assert.False(t, isValidURL("https://"), "no host")
	assert.True(t, isValidURL("http://hello"), "should be considered valid")
	assert.True(t, isValidURL("https://hello.com"), "should be considered valid")
}
