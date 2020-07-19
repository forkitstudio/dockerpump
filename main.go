package main

import (
	"encoding/json"
	"fmt"
	"github.com/forkitstudio/dockerpump/docker_client"
	"github.com/gorilla/mux"
	"io/ioutil"
	"log"
	"net/http"
	"os"
)

var sourceSrv = ""
var targetSrv = ""

const contentType = "application/json"

type HTTPResponse struct {
	Status  bool        `json:"status"`
	Cause   error       `json:"error"`
	Details interface{} `json:"details"`
}

// ResponseBody returns JSON response body.
func (r *HTTPResponse) ResponseBody() ([]byte, error) {
	body, err := json.MarshalIndent(r, "", "    ")
	fmt.Println(string(body))
	if err != nil {
		return nil, fmt.Errorf("Error while parsing response body: %v", err)
	}
	return body, nil
}

func NewHTTPResponse(err error, status bool, details string) *HTTPResponse {
	return &HTTPResponse{
		Cause:   err,
		Details: details,
		Status:  status,
	}
}

func health(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	deepCheck := vars["deep"]

	w.Header().Set("Content-Type", contentType)

	if "true" == deepCheck {
		ping, err := docker_client.Health()
		if err != nil {
			body, _ := NewHTTPResponse(err, false, "Error occurred while checking docker connectivity").ResponseBody()
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write(body)
			return
		}
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(ping)
		return
	}
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]bool{"ok": true})
}

func copyImage(w http.ResponseWriter, r *http.Request) {
	cleanupStore := true
	vars := mux.Vars(r)
	cleanup := vars["cleanup"]
	if "false" == cleanup {
		cleanupStore = false
	}

	w.Header().Set("Content-Type", contentType)

	reqBody, _ := ioutil.ReadAll(r.Body)
	var dockerImage docker_client.DockerImage
	err := json.Unmarshal(reqBody, &dockerImage)
	if err != nil {
		body, _ := NewHTTPResponse(err, false, "Can't parse request").ResponseBody()
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write(body)
		return
	}

	err = docker_client.CopyImage(sourceSrv, targetSrv, dockerImage, cleanupStore)
	if err != nil {
		//http.Error()
		body, _ := NewHTTPResponse(err, false, "Error occurred while copying image").ResponseBody()
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write(body)
		return
	}

	w.WriteHeader(http.StatusOK)
	body, _ := NewHTTPResponse(nil, true, "").ResponseBody()
	_, _ = w.Write(body)
}

func handleRequests() {
	router := mux.NewRouter().StrictSlash(true)
	router.HandleFunc("/api/health", health)
	router.HandleFunc("/api/copy_image", copyImage).Methods("POST")
	log.Fatal(http.ListenAndServe(":10000", router))
}

func main() {
	_sourceSrv, _ := os.LookupEnv("DOCKER_REGISTRY_SOURCE_SERVER")
	_targetSrv, exists := os.LookupEnv("DOCKER_REGISTRY_TARGET_SERVER")
	log.Printf("DOCKER_REGISTRY_SOURCE_SERVER: %s", _sourceSrv)
	log.Printf("DOCKER_REGISTRY_TARGET_SERVER: %s", _targetSrv)
	if !exists {
		panic("Variable DOCKER_REGISTRY_TARGET_SERVER is empty. Target registry server does not specified!")
	}
	sourceSrv = _sourceSrv
	targetSrv = _targetSrv

	handleRequests()
}
