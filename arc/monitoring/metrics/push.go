package main

import (
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/golang/snappy"
	"github.com/prometheus/prometheus/prompb"
	"github.com/golang/protobuf/proto"
)

func main() {
	http.HandleFunc("/push", PushMetricsToGeneva)

	err := http.ListenAndServe(":8090", nil)
	if err != nil {
		fmt.Printf("ListenAndServe returned error: %s", err)
	}
}

// PushMetricsToGeneva is the handler for the /push endpoint which forwards the metrics to Geneva.
func PushMetricsToGeneva(w http.ResponseWriter, r *http.Request) {
	fmt.Println("Pushing metrics")
	requestBody, err := ioutil.ReadAll(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer r.Body.Close()

	decodedRequest, err := snappy.Decode(nil, requestBody)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
	}

	writeRequest := prompb.WriteRequest{}
	if err := proto.Unmarshal(decodedRequest, &writeRequest); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	fmt.Printf("Received %d timeseries...", len(writeRequest.Timeseries))
	w.WriteHeader(http.StatusAccepted)
}
