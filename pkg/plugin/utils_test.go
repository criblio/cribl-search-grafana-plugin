package plugin

import (
	"fmt"
	"testing"
	"time"

	"github.com/criblcloud/search-datasource/pkg/models"
	"github.com/grafana/grafana-plugin-sdk-go/data"
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

func TestPrepareQuery(t *testing.T) {
	assert.Equal(t, "hello there dude\n// Grafana plugin", prepareQuery("hello\nthere\ndude"))
	assert.Equal(t, "hello there aw yeah\n// Grafana plugin", prepareQuery("hello\nthere\taw\r\nyeah"))
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

func TestMakeEmptyConcreteTypeArray(t *testing.T) {
	t.Run("string produces nullable string slice", func(t *testing.T) {
		arr, err := makeEmptyConcreteTypeArray("hello")
		assert.NoError(t, err)
		assert.IsType(t, []*string{}, arr)
	})

	t.Run("float64 produces nullable float64 slice", func(t *testing.T) {
		arr, err := makeEmptyConcreteTypeArray(float64(42))
		assert.NoError(t, err)
		assert.IsType(t, []*float64{}, arr)
	})

	t.Run("bool produces nullable bool slice", func(t *testing.T) {
		arr, err := makeEmptyConcreteTypeArray(true)
		assert.NoError(t, err)
		assert.IsType(t, []*bool{}, arr)
	})

	t.Run("time.Time produces nullable time slice", func(t *testing.T) {
		arr, err := makeEmptyConcreteTypeArray(time.Now())
		assert.NoError(t, err)
		assert.IsType(t, []*time.Time{}, arr)
	})

	t.Run("nil returns error", func(t *testing.T) {
		_, err := makeEmptyConcreteTypeArray(nil)
		assert.Error(t, err)
	})

	t.Run("unsupported type returns error", func(t *testing.T) {
		_, err := makeEmptyConcreteTypeArray([]interface{}{})
		assert.Error(t, err)
	})
}

func TestSafeAppendToField(t *testing.T) {
	t.Run("string field", func(t *testing.T) {
		field := data.NewField("test", nil, []*string{})

		s := "hello"
		safeAppendToField(field, "hello")
		assert.Equal(t, 1, field.Len())
		assert.Equal(t, &s, field.At(0))

		safeAppendToField(field, nil)
		assert.Equal(t, 2, field.Len())
		assert.Nil(t, field.At(1))

		// type mismatch: float64 into string field
		safeAppendToField(field, float64(42))
		assert.Equal(t, 3, field.Len())
		assert.Nil(t, field.At(2))
	})

	t.Run("float64 field", func(t *testing.T) {
		field := data.NewField("test", nil, []*float64{})

		f := float64(3.14)
		safeAppendToField(field, float64(3.14))
		assert.Equal(t, 1, field.Len())
		assert.Equal(t, &f, field.At(0))

		safeAppendToField(field, nil)
		assert.Equal(t, 2, field.Len())
		assert.Nil(t, field.At(1))

		// type mismatch: string into float64 field (the "string, not float64" panic case)
		safeAppendToField(field, "not a number")
		assert.Equal(t, 3, field.Len())
		assert.Nil(t, field.At(2))
	})

	t.Run("bool field", func(t *testing.T) {
		field := data.NewField("test", nil, []*bool{})

		b := true
		safeAppendToField(field, true)
		assert.Equal(t, 1, field.Len())
		assert.Equal(t, &b, field.At(0))

		safeAppendToField(field, nil)
		assert.Equal(t, 2, field.Len())
		assert.Nil(t, field.At(1))
	})

	t.Run("time field", func(t *testing.T) {
		field := data.NewField("test", nil, []*time.Time{})

		ts := time.Date(2026, 9, 25, 0, 0, 0, 0, time.UTC)
		safeAppendToField(field, ts)
		assert.Equal(t, 1, field.Len())
		assert.Equal(t, &ts, field.At(0))

		safeAppendToField(field, nil)
		assert.Equal(t, 2, field.Len())
		assert.Nil(t, field.At(1))
	})

	t.Run("extend still fills with nils", func(t *testing.T) {
		field := data.NewField("test", nil, []*string{})
		field.Extend(3)
		assert.Equal(t, 3, field.Len())
		for i := 0; i < 3; i++ {
			assert.Nil(t, field.At(i))
		}
	})
}

func TestFlattenNestedObjectToString(t *testing.T) {
	t.Run("scalars pass through unchanged", func(t *testing.T) {
		assert.Equal(t, "hello", flattenNestedObjectToString("hello"))
		assert.Equal(t, float64(42), flattenNestedObjectToString(float64(42)))
		assert.Equal(t, true, flattenNestedObjectToString(true))
		assert.Nil(t, flattenNestedObjectToString(nil))
	})

	t.Run("map is flattened to JSON string", func(t *testing.T) {
		result := flattenNestedObjectToString(map[string]interface{}{"key": "value"})
		assert.Equal(t, `{"key":"value"}`, result)
	})

	t.Run("nested map is flattened to JSON string", func(t *testing.T) {
		result := flattenNestedObjectToString(map[string]interface{}{
			"outer": map[string]interface{}{"inner": 42},
		})
		assert.IsType(t, "", result)
		assert.Contains(t, result.(string), `"outer"`)
		assert.Contains(t, result.(string), `"inner"`)
	})

	t.Run("array is flattened to JSON string", func(t *testing.T) {
		result := flattenNestedObjectToString([]interface{}{"foo", "bar"})
		assert.Equal(t, `["foo","bar"]`, result)
	})

	t.Run("array of mixed types is flattened to JSON string", func(t *testing.T) {
		result := flattenNestedObjectToString([]interface{}{"hello", float64(42), true, nil})
		assert.Equal(t, `["hello",42,true,null]`, result)
	})

	t.Run("empty array is flattened to JSON string", func(t *testing.T) {
		result := flattenNestedObjectToString([]interface{}{})
		assert.Equal(t, `[]`, result)
	})

	t.Run("empty map is flattened to JSON string", func(t *testing.T) {
		result := flattenNestedObjectToString(map[string]interface{}{})
		assert.Equal(t, `{}`, result)
	})
}

func TestIsLocalDevelopmentURL(t *testing.T) {
	assert.False(t, isLocalDevelopmentURL(""), "empty string")
	assert.False(t, isLocalDevelopmentURL("invalid url"), "invalid URL")
	assert.False(t, isLocalDevelopmentURL("https://example.com"), "regular domain")
	assert.False(t, isLocalDevelopmentURL("https://example.cribl.cloud"), "cloud domain")
	assert.False(t, isLocalDevelopmentURL("https://192.168.1.1"), "IP address")
	assert.False(t, isLocalDevelopmentURL("https://my-server.local"), "local domain")
	assert.True(t, isLocalDevelopmentURL("http://localhost"), "localhost http")
	assert.True(t, isLocalDevelopmentURL("https://localhost"), "localhost https")
	assert.True(t, isLocalDevelopmentURL("https://localhost:9000"), "localhost with port")
	assert.True(t, isLocalDevelopmentURL("http://host.docker.internal"), "docker internal http")
	assert.True(t, isLocalDevelopmentURL("https://host.docker.internal"), "docker internal https")
	assert.True(t, isLocalDevelopmentURL("https://host.docker.internal:9000"), "docker internal with port")
}
