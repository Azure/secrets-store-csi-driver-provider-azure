package converter

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"strings"
	"time"

	"github.com/Azure/secrets-store-csi-driver-provider-azure/arc/monitoring/metrics/log"
	"github.com/Azure/secrets-store-csi-driver-provider-azure/arc/monitoring/metrics/statsd"

	"github.com/prometheus/prometheus/prompb"
)

var (
	statsdClient *statsd.Client
	logger       *log.Logger
	port         = getenv("SERVER_PORT")
	// region            = getenv("REGION")
	// underlayName      = getenv("UNDERLAY")
	statsdHost     = getenvD("STATSD_ENDPOINT", "localhost")
	statsdPort     = getenvD("STATSD_PORT", "8125")
	statsdProtocol = getenvD("STATSD_PROTOCOL", "udp")
	defaultMdmNs   = getenvD("GENEVA_DEFAULT_MDM_NAMESPACE", "MetricTutorial")
	statsdEndpoint = fmt.Sprintf("%s:%s", statsdHost, statsdPort)
	// rulesFileLocation = pflag.String("rules", "/config/rules.yaml", "rules file to filter metrics/dimensions")
)

const (
	regionDim   = "Region"
	underlayDim = "UnderlayName"
)

type metricMetadata struct {
	Namespace string            `json:"Namespace"`
	Metric    string            `json:"Metric"`
	Dims      map[string]string `json:"Dims"`
	Timestamp string            `json:"TS"`
}

func init() {
	logger = log.GetContextlessTraceLogger()
}

// func main() {
// 	pflag.Parse()
// 	if port == "" ||
// 		region == "" ||
// 		underlayName == "" ||
// 		statsdHost == "" ||
// 		statsdPort == "" ||
// 		defaultMdmNs == "" {
// 		// Required variables not set
// 		logger.TraceFatalf("Required variables not set")
// 	}

// 	rulesBytes, err := ioutil.ReadFile(*rulesFileLocation)
// 	if err != nil {
// 		logger.TraceFatalf("Could not open rules file: %s", err)
// 	}
// 	if err = LoadRules(rulesBytes); err != nil {
// 		logger.TraceFatalf("Could not load rules file: %s", err)
// 	}

// 	logger.TraceInfof("Default geneva metrics namespace to be used: %s", defaultMdmNs)

// 	http.HandleFunc("/receive", ReadPrometheusSendMDM)

// 	logger.TraceInfof("Opening connection to statsd/mdm collector: %s, protocol: %s", statsdEndpoint, statsdProtocol)
// 	statsdClient, err = statsd.New(statsd.Address(statsdEndpoint), statsd.Network(statsdProtocol))
// 	if err != nil {
// 		logger.TraceFatalf("Failed to connect to statsd/mdm collector: %s", err.Error())
// 	}

// 	logger.TraceInfof("Starting server at :%s", port)

// 	strPort := ":" + port

// 	err = http.ListenAndServe(strPort, nil)
// 	if err != nil {
// 		logger.TraceFatalf("ListenAndServe returned error: %s", err)
// 	}
// }

func extractMetadata(ts *prompb.TimeSeries) *metricMetadata {
	metricMetadata := &metricMetadata{
		Namespace: defaultMdmNs,
		Dims:      make(map[string]string),
	}

	// The application can send a GenevaMdmMetricName or MdmNamespace label and the converter will replace them
	genevaMdmMetricName := ""
	genevaMdmNamespace := ""

	for _, label := range ts.Labels {
		if label != nil && strings.TrimSpace(label.Value) != "" {
			if label.Name == "GenevaMdmMetricName" {
				genevaMdmMetricName = label.Value
			} else if label.Name == "MdmNamespace" {
				genevaMdmNamespace = label.Value
			} else if label.Name == "__name__" {
				metricMetadata.Metric = label.Value
			} else {
				metricMetadata.Dims[label.Name] = label.Value
			}
		}
	}

	if genevaMdmMetricName != "" {
		metricMetadata.Metric = genevaMdmMetricName
	}
	if genevaMdmNamespace != "" {
		metricMetadata.Namespace = genevaMdmNamespace
	}

	// metricMetadata.Dims[regionDim] = region
	// metricMetadata.Dims[underlayDim] = underlayName

	return metricMetadata
}

// // ReadPrometheusSendMDM extracts prometheus samples from incoming request and sends them to statsd
// func ReadPrometheusSendMDM(w http.ResponseWriter, r *http.Request) {
// 	compressed, err := ioutil.ReadAll(r.Body)
// 	defer func() {
// 		if err := r.Body.Close(); err != nil {
// 			logger.TraceError(err.Error())
// 		}
// 	}()

// 	if err != nil {
// 		logger.TraceError(err.Error())
// 		http.Error(w, err.Error(), http.StatusInternalServerError)
// 		return
// 	}

// 	reqBuf, err := snappy.Decode(nil, compressed)
// 	if err != nil {
// 		logger.TraceError(err.Error())
// 		http.Error(w, err.Error(), http.StatusBadRequest)
// 		return
// 	}

// 	var req prompb.WriteRequest
// 	if err := proto.Unmarshal(reqBuf, &req); err != nil {
// 		logger.TraceError(err.Error())
// 		http.Error(w, err.Error(), http.StatusBadRequest)
// 		return
// 	}

// 	logger.TraceDebugf("Received %d timeseries...", len(req.Timeseries))
// 	w.WriteHeader(http.StatusAccepted)

// 	totalSent := 0
// 	for _, ts := range req.Timeseries {
// 		metadata := extractMetadata(ts)

// 		if filterMetadata(metadata) {

// 			for _, v := range ts.Samples {
// 				intValue, skip := ToMdmFriendlyInt(v.Value)

// 				if !skip {
// 					metadataBytes := renderMetadata(metadata, v)
// 					statsdClient.Gauge(metadataBytes, intValue)
// 					totalSent++
// 				} else {
// 					logger.TraceDebugf("Skipped value '%v' for metric '%s'", v.Value, metadata.Metric)
// 				}
// 			}
// 		} else {
// 			logger.TraceDebugf("Sending disabled for metric: %s", metadata.Metric)
// 		}
// 	}
// 	if totalSent > 0 {
// 		logger.TraceDebugf("Sent %d events", totalSent)
// 	}
// }

// PushMetrics sends a converts prometheus metric to MDM format and sends it to geneva
func PushMetrics(writeRequest prompb.WriteRequest) {
	totalSent := 0
	for _, ts := range writeRequest.Timeseries {
		metadata := extractMetadata(ts)

		for _, v := range ts.Samples {
			intValue, skip := ToMdmFriendlyInt(v.Value)

			if !skip {
				metadataBytes := renderMetadata(metadata, &v)
				statsdClient.Gauge(metadataBytes, intValue)
				totalSent++
			} else {
				logger.TraceDebugf("Skipped value '%v' for metric '%s'", v.Value, metadata.Metric)
			}
		}

		// if filterMetadata(metadata) {

		// 	for _, v := range ts.Samples {
		// 		intValue, skip := ToMdmFriendlyInt(v.Value)

		// 		if !skip {
		// 			metadataBytes := renderMetadata(metadata, &v)
		// 			statsdClient.Gauge(metadataBytes, intValue)
		// 			totalSent++
		// 		} else {
		// 			logger.TraceDebugf("Skipped value '%v' for metric '%s'", v.Value, metadata.Metric)
		// 		}
		// 	}
		// } else {
		// 	logger.TraceDebugf("Sending disabled for metric: %s", metadata.Metric)
		// }
	}
	if totalSent > 0 {
		logger.TraceDebugf("Sent %d events", totalSent)
	}
}

func renderMetadata(metadata *metricMetadata, sample *prompb.Sample) string {
	timestamp := time.Unix(0, sample.Timestamp*1000000) // Prometheus stores timestamps in milliseconds since epoch
	metadata.Timestamp = timestamp.UTC().Format("2006-01-02T15:04:05.000")
	metadataBytes, err := json.Marshal(metadata)
	if err != nil {
		logger.TraceErrorf("Failed to marshal metadata: Error=%s\n", err)
	}

	logger.TraceDebugf("Metric: %s", string(metadataBytes))
	return string(metadataBytes)
}

// ToMdmFriendlyInt converts float values to int64
func ToMdmFriendlyInt(value float64) (result int64, skip bool) {
	if math.IsInf(value, 1.0) {
		return math.MaxInt64, false
	}
	if math.IsInf(value, -1.0) {
		return math.MinInt64, false
	}
	if math.IsNaN(value) {
		return 0, true
	}
	intValue := int64(math.Round(value))
	if intValue == math.MinInt64 ||
		intValue == math.MaxInt64 {
		intValue = 0
	}
	return intValue, false
}

// // if filterMetadata returns true, samples should be sent
// // if filterMetadata returns false, samples shouldn't be sent
// // if # of dimensions >=50, samples won't be sent (statsd limitation)
// func filterMetadata(metadata *metricMetadata) bool {
// 	// if !isMetricAllowed(metadata.Metric) {
// 	// 	return false
// 	// }

// 	for key := range metadata.Dims {
// 		if !isDimensionAllowed(metadata.Metric, key) {
// 			delete(metadata.Dims, key)
// 		}
// 	}

// 	if len(metadata.Dims) >= 50 {
// 		tooManyDimsError := fmt.Sprintf("%s has too many dimensions. Sent: %d.", metadata.Metric, len(metadata.Dims))
// 		logger.TraceErrorf(tooManyDimsError)
// 		return false
// 	}

// 	return true
// }

func getenvD(key, fallback string) string {
	value := os.Getenv(key)
	if len(value) == 0 {
		return fallback
	}
	return value
}

func getenv(key string) string {
	value := os.Getenv(key)
	return value
}
