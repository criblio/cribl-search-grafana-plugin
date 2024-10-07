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

func TestTimeToIsoString(t *testing.T) {
	for _, test := range []struct {
		In       interface{}
		Expected string
	}{
		{
			In:       nil,
			Expected: "",
		},
		{
			In:       false,
			Expected: "",
		},
		{
			In:       true,
			Expected: "",
		},
		{
			In:       "whatever",
			Expected: "",
		},
		{
			In:       float64(1728744793),
			Expected: "2024-10-12T14:53:13Z",
		},
		{
			In:       float64(1728744793.123),
			Expected: "2024-10-12T14:53:13.123Z",
		},
	} {
		ok, out := timeToIsoString(test.In)
		if len(test.Expected) == 0 {
			assert.Equal(t, false, ok, fmt.Sprintf("input %v produced ok=%v, out=%v", test.In, ok, out))
		} else {
			assert.Equal(t, true, ok, fmt.Sprintf("input %v produced ok=%v, out=%v", test.In, ok, out))
			assert.Equal(t, test.Expected, out, test.In)
		}
	}
}
