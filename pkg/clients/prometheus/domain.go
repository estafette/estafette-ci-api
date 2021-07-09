package prometheus

type PrometheusQueryResponse struct {
	Status string                      `json:"status"`
	Data   PrometheusQueryResponseData `json:"data"`
}

// PrometheusQueryResponseData is used to unmarshal the response from a prometheus query
type PrometheusQueryResponseData struct {
	ResultType string                              `json:"resultType"`
	Result     []PrometheusQueryResponseDataResult `json:"result"`
}

// PrometheusQueryResponseDataResult is used to unmarshal the response from a prometheus query
type PrometheusQueryResponseDataResult struct {
	Metric interface{}   `json:"metric"`
	Value  []interface{} `json:"value"`
}
