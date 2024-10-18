package plugin

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseErrorFromResponse(t *testing.T) {
	for _, test := range []struct {
		In       string
		Expected interface{}
	}{
		{
			In:       `not even json`,
			Expected: nil,
		},
		{
			In:       `{"no":"message field"}`,
			Expected: nil,
		},
		{
			In:       `{"message":"just a message"}`,
			Expected: `just a message`,
		},
		{
			In:       `{"message":"{\"something\":\"other than a JS error\"}"}`,
			Expected: `{"something":"other than a JS error"}`,
		},
		{
			In:       `{"message":"{\"name\":\"AwesomeError\",\"message\":\"This error has no extra fields.\"}"}`,
			Expected: `AwesomeError: This error has no extra fields.`,
		},
		{
			In: `{"message":"{\"name\":\"AwesomeError\",\"message\":\"This error does have extra fields.\",\"code\":42,\"foo_bar\":false}"}`,
			Expected: []string{
				// json.Unmarshal() produces a map whose keys are in random order...can be either of these
				`AwesomeError: This error does have extra fields. (code: 42, foo_bar: false)`,
				`AwesomeError: This error does have extra fields. (foo_bar: false, code: 42)`,
			},
		},
	} {
		err := parseErrorFromResponse([]byte(test.In))
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
