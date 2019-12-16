package prometheus

import (
	"context"
	"encoding/json"
	"time"

	"fmt"
	"io/ioutil"
	"net/url"
	"strconv"

	"github.com/estafette/estafette-ci-api/config"
	foundation "github.com/estafette/estafette-foundation"
	"github.com/rs/zerolog/log"
	"github.com/sethgrid/pester"
)

// Client is the interface for communicating with prometheus
type Client interface {
	AwaitScrapeInterval(ctx context.Context)
	GetMaxMemoryByPodName(ctx context.Context, podName string) (float64, error)
	GetMaxCPUByPodName(ctx context.Context, podName string) (float64, error)
}

// NewClient creates an prometheus.Client to communicate with Prometheus
func NewClient(config config.PrometheusConfig) Client {
	return &client{
		config: config,
	}
}

type client struct {
	config config.PrometheusConfig
}

func (c *client) AwaitScrapeInterval(ctx context.Context) {
	log.Debug().Msgf("Waiting for %v seconds before querying Prometheus", c.config.ScrapeIntervalSeconds)
	time.Sleep(time.Duration(c.config.ScrapeIntervalSeconds) * time.Second)
}

func (c *client) GetMaxMemoryByPodName(ctx context.Context, podName string) (maxMemory float64, err error) {

	query := fmt.Sprintf("max_over_time(container_memory_working_set_bytes{container_name=\"estafette-ci-builder\",pod_name=\"%v\"}[3h])", podName)

	err = foundation.Retry(func() error {
		maxMemory, err = c.getQueryResult(query)
		return err
	}, foundation.DelayMillisecond(5000), foundation.Attempts(5))

	return
}

func (c *client) GetMaxCPUByPodName(ctx context.Context, podName string) (maxCPU float64, err error) {

	query := fmt.Sprintf("max_over_time(container_cpu_usage_rate1m{container_name=\"estafette-ci-builder\",pod_name=\"%v\"}[3h])", podName)

	err = foundation.Retry(func() error {
		maxCPU, err = c.getQueryResult(query)
		return err
	}, foundation.DelayMillisecond(5000), foundation.Attempts(5))

	return
}

func (c *client) getQueryResult(query string) (float64, error) {

	prometheusQueryURL := fmt.Sprintf("%v/api/v1/query?query=%v", c.config.ServerURL, url.QueryEscape(query))
	resp, err := pester.Get(prometheusQueryURL)
	if err != nil {
		return 0, fmt.Errorf("Executing prometheus query for query %v failed", query)
	}

	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return 0, fmt.Errorf("Reading prometheus query response body for query %v failed", query)
	}

	var queryResponse PrometheusQueryResponse
	if err = json.Unmarshal(body, &queryResponse); err != nil {
		return 0, fmt.Errorf("Unmarshalling prometheus query response body for query %v failed", query)
	}

	if queryResponse.Status != "success" {
		return 0, fmt.Errorf("Query response status %v for query %v not equal to 'success'", queryResponse.Status, query)
	}

	if queryResponse.Data.ResultType != "vector" {
		return 0, fmt.Errorf("Query response data result type %v for query %v not equal to 'vector'", queryResponse.Data.ResultType, query)
	}

	if len(queryResponse.Data.Result) != 1 {
		return 0, fmt.Errorf("Query response data vector length %v for query %v not equal to 1", len(queryResponse.Data.Result), query)
	}

	if len(queryResponse.Data.Result[0].Value) != 2 {
		return 0, fmt.Errorf("Query response data vector value length %v for query %v not equal to 2", len(queryResponse.Data.Result[0].Value), query)
	}

	f, err := strconv.ParseFloat(queryResponse.Data.Result[0].Value[1].(string), 64)
	if err != nil {
		return 0, err
	}

	return f, nil
}
