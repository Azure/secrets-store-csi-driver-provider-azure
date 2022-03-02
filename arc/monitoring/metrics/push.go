package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
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

	fmt.Println(string(requestBody))
	w.WriteHeader(http.StatusAccepted)
}
