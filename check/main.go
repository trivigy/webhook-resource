package main

import (
	"bufio"
	"crypto/hmac"
	"crypto/sha1"
	"crypto/tls"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"regexp"
	"strings"

	"github.com/google/go-github/github"
)

// type Check struct {
// 	Team     *string `json:"team,omitempty"`
// 	Pipeline *string `json:"pipeline,omitempty"`
// 	Resource *string `json:"resource,omitempty"`
// }

type Trigger struct {
	Event string                   `json:"event,omitempty"`
	Rules []map[string]interface{} `json:"rules,omitempty"`
	// Checks []Check                  `json:"checks,omitempty"`
}

type Source struct {
	Secret   string    `json:"secret,omitempty"`
	Insecure bool      `json:"insecure,omitempty"`
	Triggers []Trigger `json:"triggers,omitempty"`
}

type VersionIn struct {
	Ref string `json:"ref,omitempty"`
}

type VersionOut struct {
	Ref  string `json:"ref,omitempty"`
	Name string `json:"name,omitempty"`
}

type Payload struct {
	Source  Source    `json:"source,omitempty"`
	Version VersionIn `json:"version,omitempty"`
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

// func checkResource(request Request, check Check) error {
// 	u, err := url.Parse(os.Getenv("ATC_EXTERNAL_URL"))
// 	if err != nil {
// 		return err
// 	}
//
// 	u.Path = fmt.Sprintf(
// 		"/api/v1/teams/%v/pipelines/%v/resources/%v/check",
// 		check.Team,
// 		check.Pipeline,
// 		check.Resource,
// 	)
//
// 	req, err := http.NewRequest("POST", u.String(), nil)
// 	if err != nil {
// 		return err
// 	}
//
// 	req.Header.Set("Authorization", fmt.Sprintf("Bearer %v", request.Token))
// 	req.Header.Set("Content-Type", "application/json")
//
// 	c := &http.Client{}
// 	resp, err := c.Do(req)
// 	if err != nil {
// 		return err
// 	}
// 	defer resp.Body.Close()
// 	return nil
// }

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

	http.DefaultTransport.(*http.Transport).TLSClientConfig =
		&tls.Config{InsecureSkipVerify: payload.Source.Insecure}

	event, err := github.ParseWebHook(hook.Event, *hook.Payload)
	if err != nil {
		panic(err)
	}

	switch event := event.(type) {
	case *github.PushEvent:
		for _, trigger := range payload.Source.Triggers {
			switch trigger.Event {
			case "push":
				for _, rule := range trigger.Rules {
					var match github.PushEvent
					raw, err := json.Marshal(rule)
					if err != nil {
						panic(err)
					}
					if err := json.Unmarshal(raw, &match); err != nil {
						panic(err)
					}

					if match.Ref != nil {
						ptn := regexp.MustCompile(*match.Ref)
						if ptn.MatchString(*event.Ref) {
							output, err := json.Marshal([]VersionOut{{"blue", "blood"}})
							if err != nil {
								panic(err)
							}
							fmt.Print(string(output))
							return
						}
					}
				}
			}
		}
	}

	output, err := json.Marshal([]VersionOut{})
	if err != nil {
		panic(err)
	}
	fmt.Print(string(output))
}
