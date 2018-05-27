package main

import (
	"bytes"
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/gorilla/mux"
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
	Method  string     `json:"request,omitempty"`
	Path    string     `json:"path,omitempty"`
	Proto   string     `json:"proto,omitempty"`
	Headers [][]string `json:"headers,omitempty"`
	Body    []byte     `json:"body,omitempty"`
	Token   string     `json:"token,omitempty"`
}

func header(r *http.Request, key string) (string, bool) {
	if r.Header == nil {
		return "", false
	}
	if candidate := r.Header[key]; len(candidate) > 0 {
		return candidate[0], true
	}
	return "", false
}

func handleHealth(resp http.ResponseWriter, _ *http.Request) {
	resp.WriteHeader(http.StatusOK)
}

func handleProxy(w http.ResponseWriter, r *http.Request) {
	proc := time.Now()
	params := mux.Vars(r)

	var request Request
	request.Method = r.Method
	request.Path = r.URL.String()
	request.Proto = r.Proto
	request.Token = *token

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
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	ref, err := json.Marshal(request)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	encoded, err := json.Marshal(Version{From{base64.StdEncoding.EncodeToString(ref)}})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	u, err := url.Parse(*concourseUrl)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	u.Path = fmt.Sprintf(
		"/api/v1/teams/%v/pipelines/%v/resources/%v/check",
		params["team"],
		params["pipeline"],
		params["resource"],
	)

	http.DefaultTransport.(*http.Transport).TLSClientConfig =
		&tls.Config{InsecureSkipVerify: *insecure}

	req, err := http.NewRequest("POST", u.String(), bytes.NewBuffer(encoded))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %v", *token))
	req.Header.Set("Content-Type", "application/json")

	c := &http.Client{}
	resp, err := c.Do(req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		w.WriteHeader(resp.StatusCode)
		return
	}

	addr := req.RemoteAddr
	if ip, found := header(req, "X-Forwarded-For"); found {
		addr = ip
	}
	log.Printf("[%s] %.3f %d %s %s",
		addr,
		time.Now().Sub(proc).Seconds(),
		http.StatusOK,
		req.Method,
		req.URL,
	)
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
