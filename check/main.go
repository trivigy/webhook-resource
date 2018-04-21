package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
)

type Check struct {
	Team     string `json:"team"`
	Pipeline string `json:"pipeline"`
	Resource string `json:"resource"`
	Token    string `json:"token"`
}

type Checks []Check

type Source struct {
	Insecure bool   `json:"insecure"`
	Checks   Checks `json:"checks"`
}

type From struct {
	Ref string `json:"ref"`
}

type Version struct {
	From From `json:"from"`
}

type Payload struct {
	Source  Source  `json:"source"`
	Version Version `json:"version"`
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
		fmt.Fprintln(os.Stderr, "reading standard input:", err)
	}

	output, err := json.Marshal(payload)
	if err != nil {
		panic(err)
	}
	panic(string(output))

	// http.DefaultTransport.(*http.Transport).TLSClientConfig =
	// 	&tls.Config{InsecureSkipVerify: payload.Source.Insecure}
	//
	// for _, check := range payload.Source.Checks {
	// 	u, err := url.Parse(os.Getenv("ATC_EXTERNAL_URL"))
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
	fmt.Fprintf(os.Stdout, string(output))
}
