package plugin

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/criblcloud/search-datasource/pkg/models"
	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/backend/instancemgmt"
	"github.com/grafana/grafana-plugin-sdk-go/backend/resource/httpadapter"
	"github.com/grafana/grafana-plugin-sdk-go/data"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// Ensure it actually implements the interfaces we need it to
var (
	_ backend.QueryDataHandler    = (*Datasource)(nil)
	_ backend.CheckHealthHandler  = (*Datasource)(nil)
	_ backend.CallResourceHandler = (*Datasource)(nil)
)

const MAX_RESULTS = 10000 // same as what the actual Cribl UI imposes
const QUERY_PAGE_SIZE = 1000
const CRIBL_TIME_FIELD = "_time"
const MAX_BACKOFF_DURATION = 2 * time.Second
const GRAFANA_TIME_FIELD_NAME = "Time"

// Expose a counter metric tracking the # of queries, broken down by type (adhoc vs. savedSearchId)
var queryCounter = promauto.NewCounterVec(
	prometheus.CounterOpts{
		Namespace: "grafana_plugin", // recommended by Grafana in their "Best Practices" section
		Name:      "cribl_search_queries_total",
		Help:      "Total number of queries.",
	},
	[]string{"query_type"},
)

// Expose a counter metric tracking the total # of result events produced by queries
var resultsCounter = promauto.NewCounterVec(
	prometheus.CounterOpts{
		Namespace: "grafana_plugin",
		Name:      "cribl_search_results_total",
		Help:      "Total number of events in results.",
	},
	[]string{"query_type"},
)

type Datasource struct {
	ResourceHandler backend.CallResourceHandler
	Settings        *models.PluginSettings
	SearchAPI       *SearchAPI
}

// NewDatasource creates a new datasource instance.
func NewDatasource(ctx context.Context, settings backend.DataSourceInstanceSettings) (instancemgmt.Instance, error) {
	ps, err := models.LoadPluginSettings(settings)
	if err != nil {
		return nil, err
	}
	ds := &Datasource{}
	ds.Settings = ps
	ds.SearchAPI = NewSearchAPI(ps)

	mux := http.NewServeMux()
	mux.HandleFunc("/savedSearchIds", ds.handleSavedSearchIds)
	ds.ResourceHandler = httpadapter.New(mux)

	return ds, nil
}

// QueryData handles multiple queries and returns multiple responses.
// req contains the queries []DataQuery (where each query contains RefID as a unique identifier).
// The QueryDataResponse contains a map of RefID to the response for each query, and each response
// contains Frames ([]*Frame).
func (d *Datasource) QueryData(ctx context.Context, req *backend.QueryDataRequest) (*backend.QueryDataResponse, error) {
	// create response struct
	response := backend.NewQueryDataResponse()

	// loop over queries and execute them individually.
	for _, q := range req.Queries {
		res := d.query(ctx, req.PluginContext, q)

		// save the response in a hashmap
		// based on with RefID as identifier
		response.Responses[q.RefID] = res
	}

	return response, nil
}

func (d *Datasource) query(_ context.Context, _ backend.PluginContext, dataQuery backend.DataQuery) backend.DataResponse {
	var criblQuery models.CriblQuery
	if err := json.Unmarshal(dataQuery.JSON, &criblQuery); err != nil {
		return backend.ErrDataResponse(backend.StatusBadRequest, fmt.Sprintf("failed to unmarshal CriblQuery: %v", err.Error()))
	}
	backend.Logger.Debug("query", "criblQuery", criblQuery)

	// Construct an empty frame.  If we have to bail early, it's ready to go.  Normally we'll add fields/rows below.
	var response backend.DataResponse
	frame := data.NewFrame("results")
	frame.RefID = dataQuery.RefID
	response.Frames = append(response.Frames, frame)

	if err := canRunQuery(&criblQuery); err != nil {
		backend.Logger.Debug("can't run query", "err", err)
		return response // just return the empty response
	}

	// Increment the counter metric for this query type
	queryCounter.WithLabelValues(criblQuery.Type).Inc()

	earliest := dataQuery.TimeRange.From.Unix()
	latest := dataQuery.TimeRange.To.Unix()

	queryParams := url.Values{}
	if criblQuery.Type == "adhoc" {
		// In the near future, clients won't need to prepend "cribl" but as of now they still do
		queryParams.Set("query", prependCriblOperator(collapseToSingleLine(criblQuery.Query)))
		queryParams.Set("earliest", strconv.FormatInt(earliest, 10))
		queryParams.Set("latest", strconv.FormatInt(latest, 10))
	} else {
		// Saved/scheduled queries have their own earliest/latest timeframe pre-defined
		queryParams.Set("queryId", criblQuery.SavedSearchId)
	}
	backend.Logger.Debug("running query", "queryParams", queryParams)

	eventCount := 0
	totalEventCount := -1
	maxQueryDuration := time.Duration(0)
	if d.Settings.QueryTimeoutSec != nil {
		maxQueryDuration = time.Duration(*d.Settings.QueryTimeoutSec * 1e9)
	}
	backend.Logger.Info("timeout will be", "maxQueryDuration", maxQueryDuration, "queryTimeoutSec", d.Settings.QueryTimeoutSec)
	startTime := time.Now()

	// Load the search results, paging through until we've hit MAX_RESULTS or read all events, whatever comes first
	a, b := 100*time.Millisecond, 100*time.Millisecond // for Fibonacci backoff
	for {
		queryParams.Set("offset", strconv.Itoa(eventCount))
		queryParams.Set("limit", strconv.Itoa(MAX_RESULTS))

		result, err := d.SearchAPI.RunQueryAndGetResults(&queryParams)
		if err != nil {
			backend.Logger.Debug("query failed", "err", err)
			return backend.ErrDataResponse(backend.StatusBadRequest, err.Error())
		}
		backend.Logger.Debug("got query response", "header", result.Header)

		job := result.Header["job"].(map[string]interface{})
		if job == nil || job["id"] == nil {
			// Never expected to happen, but just in case, let's bail to prevent a screwy loop
			return backend.ErrDataResponse(backend.StatusBadRequest, "Unexpected error: response header line has no job or job id")
		}
		jobId := job["id"].(string)
		status := job["status"].(string)

		// After the first request, start passing jobId instead of queryId.  This serves two key purposes:
		//
		// 1. Ensure we don't mix result sets from different jobs.  This could happen if a scheduled search runs right in the
		// middle of when we're paging through results.  Once we get our first response, we lock to that job id, preventing
		// wires from getting crossed.
		//
		// 2. As you'll see below, it's possible that there weren't any results yet for the referenced search, and a new job
		// may have been kicked off.  We'll need to poll until that job has finished, and we need the job ID for that anyway.

		queryParams = url.Values{}
		queryParams.Set("jobId", jobId)

		// Normally what we expect when we're simply fetching results from a job that already completed (i.e. scheduled search)
		// is isFinished=true, and we can trust totalEventCount as final.  If there were no cached results, Cribl kicks off a
		// new job, and we get isFinished=false.  When this is the case, grab the job ID and poll until the job is finished.
		if !result.Header["isFinished"].(bool) {
			elapsed := time.Since(startTime)
			// If there's a configured timeout, ensure we don't let the query run longer than that
			if maxQueryDuration > 0 && elapsed >= maxQueryDuration {
				backend.Logger.Debug("query timed out, canceling", "jobId", jobId)
				err := d.SearchAPI.CancelQuery(jobId)
				if err != nil {
					backend.Logger.Warn("failed to cancel query", "jobId", jobId, "err", err)
				}
				return backend.ErrDataResponse(backend.StatusBadRequest, fmt.Sprintf("Job %s still not finished after %v (status=%v). Consider using a scheduled search to speed this up. https://docs.cribl.io/search/scheduled-searches/", jobId, maxQueryDuration, status))
			}
			a, b = b, a+b // Fibonacci backoff
			backoffDuration := a
			if backoffDuration > MAX_BACKOFF_DURATION {
				backoffDuration = MAX_BACKOFF_DURATION
			}
			backend.Logger.Debug("query not finished, delaying/backing off", "backoffDuration", backoffDuration.String())
			time.Sleep(backoffDuration)
			continue
		}

		backend.Logger.Debug("Job finished", "jobId", jobId, "status", status)
		if status != "completed" {
			return backend.ErrDataResponse(backend.StatusBadRequest, fmt.Sprintf("Job %s ended with status %s", jobId, status))
		}

		// The job is finished, so we can trust totalEventCount now, and we can proceed with getting the results
		totalEventCount = int(result.Header["totalEventCount"].(float64))

		for _, event := range result.Events {
			// Grab the keys and values from the event and populate fields in the frame
			for fieldName, value := range event {
				if fieldName == CRIBL_TIME_FIELD {
					// _time is in seconds, and Grafana needs it in ISO format.  If the conversion is successful
					// (which it will be most of the time), we use Grafana's well-known "time" field name instead.
					// If the conversion fails, _time is something other than seconds, and it will pass-through
					// as is with the "_time" field name.
					if ok, isoString := timeToIsoString(value); ok {
						fieldName = GRAFANA_TIME_FIELD_NAME
						value = isoString
					}
				}

				// Grafana doesn't like nested objects.  Convert it to a string as needed
				value = flattenNestedObjectToString(value)

				// Establish the field if it we haven't seen it yet
				field, fieldIdx := frame.FieldByName(fieldName)
				if fieldIdx == -1 {
					arr, err := makeEmptyConcreteTypeArray(value)
					if err != nil {
						backend.Logger.Warn("unable to add field", "fieldName", fieldName, "reason", err.Error())
						continue
					}
					field = data.NewField(fieldName, nil, arr)

					if eventCount > 0 {
						field.Extend(eventCount)
					}

					backend.Logger.Debug("adding field", "fieldName", fieldName)
					frame.Fields = append(frame.Fields, field)
				}
				field.Append(value)

				// Track min/max if it's a number field
				switch f := value.(type) {
				case float64:
					cf := data.ConfFloat64(f)
					if field.Config == nil {
						field.Config = &data.FieldConfig{Min: &cf, Max: &cf}
					} else {
						if cf < *field.Config.Min {
							field.Config.Min = &cf
						}
						if cf > *field.Config.Max {
							field.Config.Max = &cf
						}
					}
				}
			}

			eventCount++
			resultsCounter.WithLabelValues(criblQuery.Type).Inc()
		}

		backend.Logger.Debug("after processing events", "totalEventCount", totalEventCount, "eventCount", eventCount, "status", status)
		if eventCount >= MAX_RESULTS || (totalEventCount != -1 && eventCount >= totalEventCount) {
			break
		}
	}

	// Grafana is strict about every field needing to have the same length (# of values).
	// If a field appeared in only some events, it may be missing values for later events.
	// Apparently sparse data causes problems for some reason.  Whatever, Grafana.  So we
	// must "extend" any sparse fields to the full length (lame, Grafana, lame).
	for _, field := range frame.Fields {
		if field.Len() < eventCount {
			backend.Logger.Debug("extending field length", "fieldName", field.Name, "len", field.Len())
			field.Extend(eventCount - field.Len())
		}
	}

	return response
}

// CheckHealth handles health checks sent from Grafana to the plugin.
// The main use case for these health checks is the test button on the
// datasource configuration page which allows users to verify that
// a datasource is working as expected.
func (d *Datasource) CheckHealth(_ context.Context, req *backend.CheckHealthRequest) (*backend.CheckHealthResult, error) {
	res := &backend.CheckHealthResult{}

	if !isValidURL(d.Settings.CriblOrgBaseUrl) {
		res.Status = backend.HealthStatusError
		res.Message = "A valid Cribl Organization URL must be supplied"
		return res, nil
	}

	// We test the data source by loading saved search IDs.  This ensures the creds
	// are valid and we'll be able to make API calls successfully.
	_, err := d.SearchAPI.LoadSavedSearchIds()
	if err != nil {
		res.Status = backend.HealthStatusError
		res.Message = err.Error()
		return res, nil
	}

	return &backend.CheckHealthResult{
		Status:  backend.HealthStatusOk,
		Message: "Your Cribl Search data source is working properly.",
	}, nil
}

func (d *Datasource) CallResource(ctx context.Context, req *backend.CallResourceRequest, sender backend.CallResourceResponseSender) error {
	return d.ResourceHandler.CallResource(ctx, req, sender)
}

func (d *Datasource) handleSavedSearchIds(w http.ResponseWriter, r *http.Request) {
	ids, err := d.SearchAPI.LoadSavedSearchIds()
	if err != nil {
		backend.Logger.Error("error loading saved search IDs", "err", err)
		return
	}
	body, _ := json.Marshal(ids)
	w.Header().Add("Content-Type", "application/json")
	w.Write(body)
	w.WriteHeader(http.StatusOK)
}
