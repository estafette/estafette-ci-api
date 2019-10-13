package prometheus

// PrometheusQueryResponse is used to unmarshal the response from a prometheus query
// {
// 	"status":"success",
// 	"data":{
// 		"resultType":"vector",
// 		"result":[
// 			{
// 				"metric":{"beta_kubernetes_io_instance_type":"n1-standard-8","beta_kubernetes_io_os":"linux","container_name":"estafette-ci-builder","pod_name":"release-estafette-estafette-ci-api-494461757800972291-7h2bv"},
// 				"value":[1570970970.472,"558219264"]
// 			}
// 		]
// 	}
// }
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
