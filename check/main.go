package main

import (
	"bufio"
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/google/go-github/github"
)

type Check struct {
	Team     string `json:"team,omitempty"`
	Pipeline string `json:"pipeline,omitempty"`
	Resource string `json:"resource,omitempty"`
}

type Rule struct {
}

type Trigger struct {
	Event  string  `json:"event,omitempty"`
	Rules  []Rule  `json:"rules,omitempty"`
	Checks []Check `json:"checks,omitempty"`
}

type Source struct {
	Secret   string    `json:"secret,omitempty"`
	Insecure bool      `json:"insecure,omitempty"`
	Triggers []Trigger `json:"triggers,omitempty"`
}

type Version struct {
	Ref string `json:"ref,omitempty"`
}

type Payload struct {
	Source  Source  `json:"source,omitempty"`
	Version Version `json:"version,omitempty"`
}

type Request struct {
	Method  string     `json:"request,omitempty"`
	Path    string     `json:"path,omitempty"`
	Proto   string     `json:"proto,omitempty"`
	Headers [][]string `json:"headers,omitempty"`
	Body    []byte     `json:"body,omitempty"`
	Token   string     `json:"token,omitempty"`
}

type Hook struct {
	Delivery  string
	Event     string
	Signature string
	Payload   *[]byte
}

func (h *Hook) computeHash(secret []byte) []byte {
	computed := hmac.New(sha1.New, secret)
	computed.Write(*h.Payload)
	return []byte(computed.Sum(nil))
}

func (h *Hook) ValidateSig(secret []byte) bool {
	const sigLen = 45 // len("sha1=") + len(hex(sha1))
	if len(h.Signature) != sigLen || !strings.HasPrefix(h.Signature, "sha1=") {
		return false
	}

	actualHash := make([]byte, 20)
	hex.Decode(actualHash, []byte(h.Signature[5:]))

	return hmac.Equal(h.computeHash(secret), actualHash)
}

func main() {
	stat, err := os.Stdin.Stat()
	if err != nil {
		panic(err)
	}

	if stat.Mode()&os.ModeNamedPipe == 0 {
		panic("stdin is empty")
	}

	var payload Payload
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		if err := json.Unmarshal(scanner.Bytes(), &payload); err != nil {
			panic(err)
		}
	}
	if err := scanner.Err(); err != nil {
		panic(err)
	}

	if !strings.HasPrefix(payload.Version.Ref, "[$(ιοο)$]") {
		output, err := json.Marshal([]Version{})
		if err != nil {
			panic(err)
		}
		fmt.Print(string(output))
		return
	}

	payload.Version.Ref = strings.TrimPrefix(payload.Version.Ref, "[$(ιοο)$]")
	decoded, err := base64.StdEncoding.DecodeString(payload.Version.Ref)
	if err != nil {
		panic(err)
	}

	var request Request
	if err := json.Unmarshal(decoded, &request); err != nil {
		panic(err)
	}

	var hook Hook
	hook.Payload = &request.Body

	for _, header := range request.Headers {
		switch header[0] {
		case "x-github-delivery":
			hook.Delivery = header[1]
		case "x-github-event":
			hook.Event = header[1]
		case "x-hub-signature":
			hook.Signature = header[1]
		}
	}

	if hook.Delivery == "" {
		panic("x-github-delivery header is missing")
	}
	if hook.Event == "" {
		panic("x-github-event header is missing")
	}
	if hook.Signature == "" {
		panic("x-hub-signature header is missing")
	}

	if !hook.ValidateSig([]byte(payload.Source.Secret)) {
		panic("Invalid signature")
	}

	event, err := github.ParseWebHook(hook.Event, *hook.Payload)
	if err != nil {
		panic(err)
	}

	switch event := event.(type) {
	case *github.PushEvent:
		fmt.Println(event.GetForced())

	case *github.PullRequestEvent:
		// this is a pull request, do something with it
		// case *github.WatchEvent:
		// 	// https://developer.github.com/v3/activity/events/types/#watchevent
		// 	// someone starred our repository
		// 	if e.Action != nil && *e.Action == "starred" {
		// 		fmt.Printf("%s starred repository %s\n",
		// 			*e.Sender.Login, *e.Repo.FullName)
		// 	}
		// default:
		// 	log.Printf("unknown event type %s\n", github.WebHookType(r))
		// 	return
	}

	output, err := json.Marshal(payload)
	if err != nil {
		panic(err)
	}
	panic(err)

	// http.DefaultTransport.(*http.Transport).TLSClientConfig =
	// 	&tls.Config{InsecureSkipVerify: payload.Source.Insecure}
	//
	// for _, check := range payload.Source.Checks {
	// 	u, err := url.GithubHook(os.Getenv("ATC_EXTERNAL_URL"))
	// 	if err != nil {
	// 		panic(err)
	// 	}
	//
	// 	if len(check.Team) == 0 {
	// 		panic("team parameter is missing")
	// 	}
	//
	// 	if len(check.Pipeline) == 0 {
	// 		panic("pipeline parameter is missing")
	// 	}
	//
	// 	if len(check.Resource) == 0 {
	// 		panic("resource parameter is missing")
	// 	}
	//
	// 	if len(check.Token) == 0 {
	// 		panic("token parameter is missing")
	// 	}
	//
	// 	u.Path = fmt.Sprintf(
	// 		"/api/v1/teams/%v/pipelines/%v/resources/%v/check/webhook",
	// 		check.Team,
	// 		check.Pipeline,
	// 		check.Resource,
	// 	)
	// 	q := u.Query()
	// 	q.Set("webhook_token", check.Token)
	// 	u.RawQuery = q.Encode()
	//
	// 	req, err := http.NewRequest("POST", u.String(), nil)
	// 	if err != nil {
	// 		panic(err)
	// 	}
	//
	// 	c := &http.Client{}
	// 	resp, err := c.Do(req)
	// 	if err != nil {
	// 		panic(err)
	// 	}
	// 	resp.Body.Close()
	// }

	output, err = json.Marshal([]Version{})
	if err != nil {
		panic(err)
	}
	fmt.Print(string(output))
}
