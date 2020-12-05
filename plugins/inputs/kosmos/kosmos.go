package kosmos

// Based on nsq input plugin

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/influxdata/telegraf"
	"github.com/influxdata/telegraf/plugins/common/tls"
	"github.com/influxdata/telegraf/plugins/inputs"
)

type Kosmos struct {
	Url string
	tls.ClientConfig
	httpClient *http.Client
}

const sampleConfig = `
  ## URL to Kosmos metrics API.
  url = "http://localhost:19092/metrics/v2/values"

  ## Optional TLS Config
  # tls_ca = "/etc/telegraf/ca.pem"
  # tls_cert = "/etc/telegraf/cert.pem"
  # tls_key = "/etc/telegraf/key.pem"
  ## Use TLS but skip chain & host verification
  # insecure_skip_verify = true
`

func (*Kosmos) SampleConfig() string {
	return sampleConfig
}

func (*Kosmos) Description() string {
	return "Collects metrics reported by the Kosmos metrics API."
}

func (v *Kosmos) Gather(acc telegraf.Accumulator) error {
	var err error

	if v.httpClient == nil {
		v.httpClient, err = v.getHttpClient()
		if err != nil {
			return err
		}
	}
	response, err := v.httpClient.Get(v.Url)
	if err != nil {
		return err
	}
	responseBody, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return err
	}
	data := &KosmosMetricResponse{}
	if err := json.Unmarshal(responseBody, data); err != nil {
		return err
	}
	if len(data.Nodes) == 0 {
		return fmt.Errorf("Kosmos metrics API returned unexpected json")
	}
	var node = data.Nodes[0]
	for _, service := range node.Services {
		for _, metrics := range service.Metrics {
			acc.AddGauge("kosmos", metrics.Values, metrics.Dimensions)
		}
	}
	for _, metrics := range node.Node.Metrics {
		acc.AddGauge("kosmos", metrics.Values, metrics.Dimensions)
	}
	return nil
}

func (v *Kosmos) getHttpClient() (*http.Client, error) {
	tlsConfig, err := v.ClientConfig.TLSConfig()
	if err != nil {
		return nil, err
	}
	tr := &http.Transport{
		TLSClientConfig: tlsConfig,
	}
	httpClient := &http.Client{
		Transport: tr,
		Timeout:   time.Duration(4 * time.Second),
	}
	return httpClient, nil
}

func init() {
	inputs.Add("kosmos", func() telegraf.Input {
		return &Kosmos{}
	})
}

type KosmosMetricResponse struct {
	Nodes []Nodes `json:"nodes"`
}

type Nodes struct {
	Hostname string    `json:"hostname"`
	Role     string    `json:"role"`
	Node     Node      `json:"node"`
	Services []Service `json:"services"`
}

type Node struct {
	Timestamp int64          `json:"timestamp"`
	Metrics   []MetricPacket `json:"metrics"`
}

type Service struct {
	Name      string         `json:"name"`
	Timestamp int64          `json:"timestamp"`
	Metrics   []MetricPacket `json:"metrics"`
}

type MetricPacket struct {
	Values     map[string]interface{} `json:"values"`
	Dimensions map[string]string      `json:"dimensions"`
}
