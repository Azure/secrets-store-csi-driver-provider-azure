package main

import (
	"math"
	"time"

	"github.com/Azure/secrets-store-csi-driver-provider-azure/arc/monitoring/metrics/log"

	"github.com/MakeNowJust/heredoc"

	"testing"

	"github.com/prometheus/prometheus/prompb"
	"github.com/stretchr/testify/assert"
)

func init() {
	logger = log.GetContextlessTraceLogger()
}

func initPrometheusSample() *prompb.TimeSeries {
	prometheusTimeseries := &prompb.TimeSeries{}

	labelMap := map[string]string{
		"__name__":                                 "container_tasks_state",
		"agentpool1":                               "nodepool1",
		"beta_kubernetes_io_arch":                  "amd64",
		"beta_kubernetes_io_instance_type":         "Standard_D1_v2",
		"beta_kubernetes_io_os":                    "linux",
		"container_name":                           "kube-proxy",
		"failure_domain_beta_kubernetes_io_region": "canadaeast",
		"failure_domain_beta_kubernetes_io_zone":   "0",
		"id":                                       "/docker/fc707d647c395c2883f65b31eeb1547d5818fd6cc29d5905dc95378774340926",
		"image":                                    "k8s.gcr.io/hyperkube-amd64@sha256:e76df5a9415301227607f52859ed79e05a70816ba331338d56b5c66c2a1e5c1c",
		"instance":                                 "aks-nodepool1-28351083-1",
		"job":                                      "kubernetes-nodes-cadvisor",
		"kubernetes_azure_com_cluster":             "MC_sgoings-caeast_sgoings-caeast_canadaeast",
		"kubernetes_io_hostname":                   "aks-nodepool1-28351083-1",
		"kubernetes_io_role":                       "agent",
		"name":                                     "k8s_kube-proxy_kube-proxy-6zsm5_kube-system_2bbc7d53-20bc-11e8-bd41-0a58ac1f13b1_0",
		"namespace":                                "kube-system",
		"pod_name":                                 "kube-proxy-6zsm5",
		"state":                                    "iowaiting",
		"storageprofile":                           "managed",
		"storagetier":                              "Standard_LRS",
	}

	labels := make([]*prompb.Label, len(labelMap))

	for k, v := range labelMap {
		newLabel := &prompb.Label{
			Name:  k,
			Value: v,
		}

		labels = append(labels, newLabel)
	}

	prometheusTimeseries.Samples = make([]prompb.Sample, 1)
	prometheusTimeseries.Samples[0] = prompb.Sample{
		Value:     0.0,
		Timestamp: 1520351553485,
	}

	prometheusTimeseries.Labels = labels

	return prometheusTimeseries
}

func Test_ConvertPrometheusMetricToMDMFormat(t *testing.T) {

	prometheusTimeseries := initPrometheusSample()
	timestamp, err := time.ParseInLocation("2006-01-02T15:04:05.000", "2016-02-03T18:02:03.456", time.UTC)
	assert.NoError(t, err)

	metadata := extractMetadata(prometheusTimeseries)
	metadataString := renderMetadata(metadata, &prompb.Sample{
		Timestamp: timestamp.UnixNano() / 1000000,
	})

	assert.Equal(t, "container_tasks_state", metadata.Metric)
	assert.Equal(t, "Prometheus", metadata.Namespace)

	expectedMetadataString := heredoc.Doc(`
			{
				"Namespace": "Prometheus",
				"Metric": "container_tasks_state",
				"Dims": {
					"Region": "",
					"UnderlayName": "",
					"agentpool1": "nodepool1",
					"beta_kubernetes_io_arch": "amd64",
					"beta_kubernetes_io_instance_type": "Standard_D1_v2",
					"beta_kubernetes_io_os": "linux",
					"container_name": "kube-proxy",
					"failure_domain_beta_kubernetes_io_region": "canadaeast",
					"failure_domain_beta_kubernetes_io_zone": "0",
					"id": "/docker/fc707d647c395c2883f65b31eeb1547d5818fd6cc29d5905dc95378774340926",
					"image": "k8s.gcr.io/hyperkube-amd64@sha256:e76df5a9415301227607f52859ed79e05a70816ba331338d56b5c66c2a1e5c1c",
					"instance": "aks-nodepool1-28351083-1",
					"job": "kubernetes-nodes-cadvisor",
					"kubernetes_azure_com_cluster": "MC_sgoings-caeast_sgoings-caeast_canadaeast",
					"kubernetes_io_hostname": "aks-nodepool1-28351083-1",
					"kubernetes_io_role": "agent",
					"name": "k8s_kube-proxy_kube-proxy-6zsm5_kube-system_2bbc7d53-20bc-11e8-bd41-0a58ac1f13b1_0",
					"namespace": "kube-system",
					"pod_name": "kube-proxy-6zsm5",
					"state": "iowaiting",
					"storageprofile": "managed",
					"storagetier": "Standard_LRS"
				},
				"TS": "2016-02-03T18:02:03.456"
			}`)

	assert.JSONEq(t, expectedMetadataString, metadataString)

	assert.Equal(t, float64(0), prometheusTimeseries.Samples[0].Value)
	assert.Equal(t, int64(1520351553485), prometheusTimeseries.Samples[0].Timestamp)
}

func assertToMdmFriendlyInt(t *testing.T, input float64, expectedValue int64, expectedSkip bool) {
	value, skip := ToMdmFriendlyInt(input)
	assert.Equal(t, expectedValue, value)
	assert.Equal(t, expectedSkip, skip)
}

func Test_ToMdmFriendlyInt(t *testing.T) {
	assertToMdmFriendlyInt(t, 123.3, int64(123), false)
	assertToMdmFriendlyInt(t, -456.7, int64(-457), false)
	assertToMdmFriendlyInt(t, math.MaxFloat64, int64(0), false)
	assertToMdmFriendlyInt(t, -1*math.MaxFloat64, int64(0), false)
	assertToMdmFriendlyInt(t, math.NaN(), int64(0), true)
	assertToMdmFriendlyInt(t, math.Inf(1.0), int64(math.MaxInt64), false)
	assertToMdmFriendlyInt(t, math.Inf(-1.0), int64(math.MinInt64), false)
}
