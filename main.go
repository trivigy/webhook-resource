package main

import (
	"flag"
	"github.com/gorilla/mux"
	"log"
	"net/http"
	"strings"
	"io/ioutil"
	"fmt"
	"encoding/json"
	"encoding/base64"
	"net/url"
	"bytes"
	"crypto/tls"
)

var (
	bind = flag.String(
		"bind",
		"127.0.0.1:8080",
		"Listening address for incoming requests",
	)
	concourseUrl = flag.String(
		"concourse-url",
		"http://localhost:8888",
		"Concourse URL to forward the reqeuest",
	)
	token = flag.String(
		"token",
		"",
		"Bearer oAuth authorization token",
	)
	insecure = flag.Bool(
		"insecure",
		false,
		"Disable ssl security verification",
	)
)

type From struct {
	Ref string `json:"ref"`
}

type Version struct {
	From From `json:"from"`
}

type Request struct {
	Method  string     `json:"request"`
	Path    string     `json:"path"`
	Proto   string     `json:"proto"`
	Headers [][]string `json:"headers"`
	Body    []byte     `json:"body"`
}

func handleHealth(resp http.ResponseWriter, _ *http.Request) {
	resp.WriteHeader(http.StatusOK)
}

func handleProxy(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)

	var request Request
	request.Method = r.Method
	request.Path = r.URL.String()
	request.Proto = r.Proto

	for name, headers := range r.Header {
		name = strings.ToLower(name)
		for _, value := range headers {
			header := []string{name, value}
			request.Headers = append(request.Headers, header)
		}
	}

	var err error
	request.Body, err = ioutil.ReadAll(r.Body)
	if err != nil {
		panic(err)
	}

	ref, err := json.Marshal(request)
	if err != nil {
		panic(err)
	}

	output, err := json.Marshal(Version{From{base64.StdEncoding.EncodeToString(ref)}})
	if err != nil {
		panic(err)
	}

	u, err := url.Parse(*concourseUrl)
	if err != nil {
		panic(err)
	}

	u.Path = fmt.Sprintf(
		"/api/v1/teams/%v/pipelines/%v/resources/%v/check",
		params["team"],
		params["pipeline"],
		params["resource"],
	)

	http.DefaultTransport.(*http.Transport).TLSClientConfig =
		&tls.Config{InsecureSkipVerify: *insecure}

	req, err := http.NewRequest("POST", u.String(), bytes.NewBuffer(output))
	if err != nil {
		panic(err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %v", token))
	req.Header.Set("Content-Type", "application/json")

	c := &http.Client{}
	resp, err := c.Do(req)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func main() {
	flag.Parse()

	router := mux.NewRouter()

	router.HandleFunc("/health", handleHealth).
		Methods("GET")

	router.HandleFunc("/hook/{team:.+}/{pipeline:.+}/{resource:.+}", handleProxy).
		Methods("POST")

	log.Printf("[service] listening on %s", *bind)
	if err := http.ListenAndServe(*bind, router); err != nil {
		log.Fatal(err)
	}
}
