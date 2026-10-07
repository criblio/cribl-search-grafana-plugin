package plugin

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"sync"
	"testing"
	"time"

	"github.com/criblcloud/search-datasource/pkg/models"
	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestQueryData(t *testing.T) {
	ds := Datasource{}

	resp, err := ds.QueryData(
		context.Background(),
		&backend.QueryDataRequest{
			Queries: []backend.DataQuery{
				{RefID: "A"},
			},
		},
	)
	if err != nil {
		t.Error(err)
	}

	if len(resp.Responses) != 1 {
		t.Fatal("QueryData must return a response")
	}
}

type mockSearchAPI struct {
	mu              sync.Mutex
	callCount       int
	cancelledJobIDs []string
	onQuery         func(ctx context.Context, callNum int, queryParams *url.Values) (*SearchQueryResult, error)
}

func (m *mockSearchAPI) RunQueryAndGetResults(ctx context.Context, queryParams *url.Values) (*SearchQueryResult, error) {
	m.mu.Lock()
	m.callCount++
	callNum := m.callCount
	m.mu.Unlock()
	if m.onQuery != nil {
		return m.onQuery(ctx, callNum, queryParams)
	}
	return &SearchQueryResult{
		Header: map[string]interface{}{
			"job":             map[string]interface{}{"id": "job1", "status": "completed"},
			"isFinished":      true,
			"totalEventCount": float64(0),
		},
	}, nil
}

func (m *mockSearchAPI) CancelQuery(ctx context.Context, jobId string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.cancelledJobIDs = append(m.cancelledJobIDs, jobId)
	return nil
}

func (m *mockSearchAPI) LoadSavedSearchIds(ctx context.Context) ([]string, error) {
	return []string{}, nil
}

func makeAdhocQuery(query string) backend.DataQuery {
	j, _ := json.Marshal(models.CriblQuery{Type: "adhoc", Query: query})
	return backend.DataQuery{
		RefID: "A",
		JSON:  j,
		TimeRange: backend.TimeRange{
			From: time.Now().Add(-1 * time.Hour),
			To:   time.Now(),
		},
	}
}

func TestQueryTimeoutOverridesGrafanaContext(t *testing.T) {
	// Simulate Grafana imposing a short 1-second context deadline while the plugin
	// has a longer 5-second query timeout configured.  The query should survive
	// past Grafana's deadline.
	timeout := 5.0
	mock := &mockSearchAPI{
		onQuery: func(ctx context.Context, callNum int, queryParams *url.Values) (*SearchQueryResult, error) {
			if callNum <= 3 {
				return &SearchQueryResult{
					Header: map[string]interface{}{
						"job":             map[string]interface{}{"id": "job1", "status": "running"},
						"isFinished":      false,
						"totalEventCount": float64(0),
					},
				}, nil
			}
			return &SearchQueryResult{
				Header: map[string]interface{}{
					"job":             map[string]interface{}{"id": "job1", "status": "completed"},
					"isFinished":      true,
					"totalEventCount": float64(0),
				},
			}, nil
		},
	}

	ds := Datasource{
		Settings: &models.PluginSettings{QueryTimeoutSec: &timeout},
		SearchAPI: mock,
	}

	// Grafana context with a very short deadline
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	resp, err := ds.QueryData(ctx, &backend.QueryDataRequest{
		Queries: []backend.DataQuery{makeAdhocQuery("test query")},
	})
	require.NoError(t, err)

	res := resp.Responses["A"]
	// The query should have completed successfully, NOT been cancelled by Grafana's 1s deadline
	assert.Empty(t, res.Error, "query should succeed despite Grafana's short context deadline")
	assert.Empty(t, mock.cancelledJobIDs, "no jobs should have been cancelled")
}

func TestQueryCancelledWhenGrafanaContextCancelled(t *testing.T) {
	// Even with a long plugin timeout, if Grafana's context is explicitly cancelled
	// (e.g. user navigates away), the query should be cancelled.
	timeout := 60.0
	queryStarted := make(chan struct{})
	mock := &mockSearchAPI{
		onQuery: func(ctx context.Context, callNum int, queryParams *url.Values) (*SearchQueryResult, error) {
			if callNum == 1 {
				close(queryStarted)
			}
			return &SearchQueryResult{
				Header: map[string]interface{}{
					"job":             map[string]interface{}{"id": "job1", "status": "running"},
					"isFinished":      false,
					"totalEventCount": float64(0),
				},
			}, nil
		},
	}

	ds := Datasource{
		Settings: &models.PluginSettings{QueryTimeoutSec: &timeout},
		SearchAPI: mock,
	}

	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan backend.DataResponse)
	go func() {
		resp, _ := ds.QueryData(ctx, &backend.QueryDataRequest{
			Queries: []backend.DataQuery{makeAdhocQuery("test query")},
		})
		done <- resp.Responses["A"]
	}()

	// Wait for the query to start, then cancel
	<-queryStarted
	cancel()

	res := <-done
	assert.Contains(t, res.Error.Error(), "Canceled", "query should report cancellation")
	assert.Equal(t, []string{"job1"}, mock.cancelledJobIDs, "the search job should have been cancelled")
}

func TestQueryTimesOutWithPluginTimeout(t *testing.T) {
	// When the plugin's query timeout is reached, the query should be cancelled with
	// a helpful message (not a generic "Query Canceled").
	timeout := 0.5 // half a second
	mock := &mockSearchAPI{
		onQuery: func(ctx context.Context, callNum int, queryParams *url.Values) (*SearchQueryResult, error) {
			return &SearchQueryResult{
				Header: map[string]interface{}{
					"job":             map[string]interface{}{"id": "job1", "status": "running"},
					"isFinished":      false,
					"totalEventCount": float64(0),
				},
			}, nil
		},
	}

	ds := Datasource{
		Settings: &models.PluginSettings{QueryTimeoutSec: &timeout},
		SearchAPI: mock,
	}

	resp, err := ds.QueryData(context.Background(), &backend.QueryDataRequest{
		Queries: []backend.DataQuery{makeAdhocQuery("test query")},
	})
	require.NoError(t, err)

	res := resp.Responses["A"]
	assert.Contains(t, res.Error.Error(), "still not finished", "should show the plugin's timeout message")
	assert.Contains(t, res.Error.Error(), "scheduled search", "should suggest scheduled search")
	assert.Equal(t, []string{"job1"}, mock.cancelledJobIDs)
}

func TestQueryNoTimeoutUsesGrafanaContext(t *testing.T) {
	// When no plugin timeout is configured, Grafana's context deadline should be respected.
	mock := &mockSearchAPI{
		onQuery: func(ctx context.Context, callNum int, queryParams *url.Values) (*SearchQueryResult, error) {
			return &SearchQueryResult{
				Header: map[string]interface{}{
					"job":             map[string]interface{}{"id": "job1", "status": "running"},
					"isFinished":      false,
					"totalEventCount": float64(0),
				},
			}, nil
		},
	}

	ds := Datasource{
		Settings: &models.PluginSettings{},
		SearchAPI: mock,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	resp, err := ds.QueryData(ctx, &backend.QueryDataRequest{
		Queries: []backend.DataQuery{makeAdhocQuery("test query")},
	})
	require.NoError(t, err)

	res := resp.Responses["A"]
	assert.Contains(t, res.Error.Error(), "Canceled", "should be cancelled by Grafana's context deadline")
	assert.Equal(t, []string{"job1"}, mock.cancelledJobIDs)
}

func TestContextPropagatedToHTTPCalls(t *testing.T) {
	// Verify that the context is passed through to RunQueryAndGetResults so that
	// in-flight HTTP calls can be interrupted.
	var receivedCtx context.Context
	mock := &mockSearchAPI{
		onQuery: func(ctx context.Context, callNum int, queryParams *url.Values) (*SearchQueryResult, error) {
			receivedCtx = ctx
			return &SearchQueryResult{
				Header: map[string]interface{}{
					"job":             map[string]interface{}{"id": "job1", "status": "completed"},
					"isFinished":      true,
					"totalEventCount": float64(0),
				},
			}, nil
		},
	}

	timeout := 30.0
	ds := Datasource{
		Settings: &models.PluginSettings{QueryTimeoutSec: &timeout},
		SearchAPI: mock,
	}

	resp, err := ds.QueryData(context.Background(), &backend.QueryDataRequest{
		Queries: []backend.DataQuery{makeAdhocQuery("test query")},
	})
	require.NoError(t, err)
	assert.Empty(t, resp.Responses["A"].Error)

	// The context passed to the mock should have a deadline matching the plugin's timeout
	require.NotNil(t, receivedCtx, "context should have been passed to SearchAPI")
	deadline, ok := receivedCtx.Deadline()
	assert.True(t, ok, "context should have a deadline from the plugin's queryTimeoutSec")
	assert.WithinDuration(t, time.Now().Add(30*time.Second), deadline, 5*time.Second,
		fmt.Sprintf("deadline should be ~30s from now, got %v", deadline))
}
