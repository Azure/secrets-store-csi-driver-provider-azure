package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/golang/protobuf/proto"
	"github.com/golang/snappy"
	"github.com/prometheus/prometheus/prompb"
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
		fmt.Printf("Error decoding request: %s", err.Error())
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	writeRequest := prompb.WriteRequest{}
	if err := proto.Unmarshal(decodedRequest, &writeRequest); err != nil {
		fmt.Printf("Error unmarshaling request: %s", err.Error())
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	con, _ := json.MarshalIndent(writeRequest, "", "  ")
	fmt.Printf("\n================TS===================\n%s\n", con)

	fmt.Printf("Received %d timeseries...\n", len(writeRequest.Timeseries))
	fmt.Println("Pushed metrics complete")
	w.WriteHeader(http.StatusAccepted)
}
