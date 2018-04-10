package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	"github.com/pkg/errors"
)

var FormatSpringHealth string = "spring-health"

type HttpCheck struct {
	environmentId string
	checkId       string
	name          string
	timeout       time.Duration
	url           string
	user          string
	password      string
	format        string
	contains      string
	header        map[string]string
}

func NewHttpCheck(environmentId, checkId, name string, params map[string]string) (*HttpCheck, error) {
	c := &HttpCheck{
		environmentId: environmentId,
		checkId:       checkId,
		name:          name,
		url:           params["url"],
		user:          params["user"],
		password:      params["password"],
		format:        params["format"],
		contains:      params["contains"],
		header:        map[string]string{},
	}
	if t, exist := params["timeout"]; exist {
		d, err := time.ParseDuration(t)
		if err != nil {
			return nil, err
		}
		c.timeout = d
	} else {
		c.timeout = time.Second * 10
	}
	for k, v := range params {
		if strings.HasPrefix(k, "header-") {
			c.header[strings.TrimPrefix(k, "header-")] = v
		}
	}
	return c, nil
}

func (c *HttpCheck) Check() []Result {
	mainResult := NewResult(c.environmentId, c.checkId, c.name)

	mainResult.Status, mainResult.Message, mainResult.Detail = c.execute()

	mainResult.Duration = int(time.Since(mainResult.Timestamp) / time.Millisecond)

	results := []Result{mainResult}

	return results
}

func (c *HttpCheck) execute() (status, message, detail string) {
	r, err := http.NewRequest("GET", c.url, nil)

	if err != nil {
		return StatusDown, err.Error(), ""
	}

	r.Header.Set("User-Agent", "statuspage")
	for k, v := range c.header {
		r.Header.Set(k, v)
	}
	if c.user != "" {
		r.SetBasicAuth(c.user, c.password)
	}

	client := http.Client{
		Timeout: c.timeout,
	}
	resp, err := client.Do(r)

	if err != nil {
		return StatusDown, err.Error(), ""
	}

	if 200 != resp.StatusCode {
		return StatusDown, fmt.Sprintf("http status code: %v\n", resp.StatusCode), ""
	}

	b, err := c.readBody(resp)
	if err != nil {
		return StatusDown, fmt.Sprintf("could not read body: %v\n", err), ""
	}

	if c.format == FormatSpringHealth {
		status, err := ensureSpringHealthFormat(b, resp)
		if err != nil {
			return StatusDown, err.Error(), string(b)
		}
		return status, "", ""
	}

	if c.contains != "" && !strings.Contains(string(b), c.contains) {
		return StatusDown, fmt.Sprintf("missing string %q in result", c.contains), string(b)
	}

	return StatusUp, "", ""
}

func (c *HttpCheck) readBody(resp *http.Response) ([]byte, error) {
	if resp.Body == nil {
		return []byte{}, nil
	}

	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return b, err
	}

	err = resp.Body.Close()
	return b, err
}

func ensureSpringHealthFormat(body []byte, resp *http.Response) (string, error) {
	if !(strings.HasPrefix(resp.Header.Get("Content-Type"), "application/json") ||
		strings.HasPrefix(resp.Header.Get("Content-Type"), "application/vnd.spring-boot.actuator")) {
		return StatusDown, fmt.Errorf("got wrong content type: %v", resp.Header.Get("Content-Type"))
	}

	resultData := map[string]interface{}{}
	err := json.Unmarshal(body, &resultData)
	if err != nil {
		return StatusDown, errors.Wrap(err, "error parsing json body")
	}
	s, exist := resultData["status"]
	if !exist {
		return StatusDown, errors.New("missing status in response")
	}
	return fmt.Sprintf("%v", s), nil
}
