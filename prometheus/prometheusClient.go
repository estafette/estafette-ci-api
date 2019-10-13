package prometheus

import (
	"encoding/json"

	"strconv"
	"fmt"
	"io/ioutil"
	"net/url"
	"github.com/estafette/estafette-ci-api/config"
	"github.com/sethgrid/pester"
)

// PrometheusClient is the interface for communicating with prometheus
type PrometheusClient interface {
	GetMaxMemoryByPodName(podName string) (float64, error)
	GetMaxCPUByPodName(podName string) (float64, error)
}

type prometheusClientImpl struct {
	config config.PrometheusConfig
}

// NewPrometheusClient creates an prometheus.PrometheusClient to communicate with Prometheus
func NewPrometheusClient(config config.PrometheusConfig) PrometheusClient {
	return &prometheusClientImpl{
		config:                          config,
	}
}

func (pc *prometheusClientImpl) GetMaxMemoryByPodName(podName string) (float64, error) {
	query := fmt.Sprintf("max_over_time(container_memory_working_set_bytes{container_name=\"estafette-ci-builder\",pod_name=\"%v\"}[3h])", podName)
	return pc.getQueryResult(query)
}

func (pc *prometheusClientImpl) GetMaxCPUByPodName(podName string) (float64, error) {
	query := fmt.Sprintf("max_over_time(container_cpu_usage_irate{container_name=\"estafette-ci-builder\",pod_name=\"%v\"}[3h])", podName)
	return pc.getQueryResult(query)
}

func (pc *prometheusClientImpl) getQueryResult(query string) (float64, error) {

	prometheusQueryURL := fmt.Sprintf("%v/api/v1/query?query=%v", pc.config.ServerURL, url.QueryEscape(query))
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