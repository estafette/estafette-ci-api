package prometheus

import (
	"context"
	"encoding/json"
	"io"
	"time"

	"fmt"
	"net/url"
	"strconv"

	"github.com/estafette/estafette-ci-api/pkg/api"
	foundation "github.com/estafette/estafette-foundation"
	"github.com/rs/zerolog/log"
	"github.com/sethgrid/pester"
)

// Client is the interface for communicating with prometheus
//
//go:generate mockgen -package=prometheus -destination ./mock.go -source=client.go
type Client interface {
	AwaitScrapeInterval(ctx context.Context)
	GetMaxMemoryByPodName(ctx context.Context, podName string) (max float64, err error)
	GetMaxCPUByPodName(ctx context.Context, podName string) (max float64, err error)
}

// NewClient creates an prometheus.Client to communicate with Prometheus
func NewClient(config *api.APIConfig) Client {
	if config == nil || config.Integrations == nil || config.Integrations.Prometheus == nil || config.Integrations.Prometheus.Enable == nil || !*config.Integrations.Prometheus.Enable {
		return &client{
			enabled: false,
		}
	}

	return &client{
		enabled: true,
		config:  config,
	}
}

type client struct {
	enabled bool
	config  *api.APIConfig
}

func (c *client) AwaitScrapeInterval(ctx context.Context) {
	if !c.enabled {
		return
	}

	log.Debug().Msgf("Waiting for %v seconds before querying Prometheus", c.config.Integrations.Prometheus.ScrapeIntervalSeconds)
	time.Sleep(time.Duration(c.config.Integrations.Prometheus.ScrapeIntervalSeconds) * time.Second)
}

func (c *client) GetMaxMemoryByPodName(ctx context.Context, podName string) (maxMemory float64, err error) {
	if !c.enabled {
		return
	}

	query := fmt.Sprintf("max_over_time(container_memory_working_set_bytes{container=\"estafette-ci-builder\",pod=\"%v\"}[6h])", podName)

	err = foundation.Retry(func() error {
		maxMemory, err = c.getQueryResult(query)
		return err
	}, foundation.DelayMillisecond(1000*c.config.Integrations.Prometheus.ScrapeIntervalSeconds), foundation.Attempts(10))

	return
}

func (c *client) GetMaxCPUByPodName(ctx context.Context, podName string) (maxCPU float64, err error) {
	if !c.enabled {
		return
	}

	query := fmt.Sprintf("max_over_time(container_cpu_usage_rate1m{container=\"estafette-ci-builder\",pod=\"%v\"}[6h])", podName)

	err = foundation.Retry(func() error {
		maxCPU, err = c.getQueryResult(query)
		return err
	}, foundation.DelayMillisecond(1000*c.config.Integrations.Prometheus.ScrapeIntervalSeconds), foundation.Attempts(10))

	return
}

func (c *client) getQueryResult(query string) (float64, error) {

	prometheusQueryURL := fmt.Sprintf("%v/api/v1/query?query=%v", c.config.Integrations.Prometheus.ServerURL, url.QueryEscape(query))
	resp, err := pester.Get(prometheusQueryURL)
	if err != nil {
		return 0, fmt.Errorf("Executing prometheus query url %v failed", prometheusQueryURL)
	}

	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, fmt.Errorf("Reading prometheus query response body for query url %v failed", prometheusQueryURL)
	}

	var queryResponse PrometheusQueryResponse
	if err = json.Unmarshal(body, &queryResponse); err != nil {
		return 0, fmt.Errorf("Unmarshalling prometheus query response body for query url %v failed", prometheusQueryURL)
	}

	if queryResponse.Status != "success" {
		return 0, fmt.Errorf("Query response status %v for query url %v not equal to 'success'", queryResponse.Status, prometheusQueryURL)
	}

	if queryResponse.Data.ResultType != "vector" {
		return 0, fmt.Errorf("Query response data result type %v for query url %v not equal to 'vector'", queryResponse.Data.ResultType, prometheusQueryURL)
	}

	if len(queryResponse.Data.Result) != 1 {
		return 0, fmt.Errorf("Query response data vector length %v for query url %v not equal to 1", len(queryResponse.Data.Result), prometheusQueryURL)
	}

	if len(queryResponse.Data.Result[0].Value) != 2 {
		return 0, fmt.Errorf("Query response data vector value length %v for query url %v not equal to 2", len(queryResponse.Data.Result[0].Value), prometheusQueryURL)
	}

	f, err := strconv.ParseFloat(queryResponse.Data.Result[0].Value[1].(string), 64)
	if err != nil {
		return 0, err
	}

	return f, nil
}
