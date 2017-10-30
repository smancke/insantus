package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/pkg/errors"
	"io/ioutil"
	"net/http"
	"strings"
	"time"
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
	return c, nil
}

func (c *HttpCheck) Check() []Result {
	mainResult := NewResult(c.environmentId, c.checkId, c.name)

	r, err := http.NewRequest("GET", c.url, nil)
	r.Header.Set("User-Agent", "statuspage")

	if err != nil {
		mainResult.Status = StatusError
		mainResult.Message = err.Error()
	} else {

		if c.user != "" {
			r.SetBasicAuth(c.user, c.password)
		}
		ctx, cancel := context.WithTimeout(context.Background(), c.timeout)
		defer cancel()
		r.WithContext(ctx)

		resp, err := http.DefaultClient.Do(r)
		if err != nil {
			mainResult.Status = StatusError
			mainResult.Message = err.Error()
		} else {
			if 200 != resp.StatusCode {
				mainResult.Status = StatusDown
				mainResult.Message = fmt.Sprintf("http status code: %v\n", resp.StatusCode)
			}
			fmt.Println("contains: ", c.contains)

			if c.format == FormatSpringHealth ||
				c.contains != "" {

				b, _ := ioutil.ReadAll(resp.Body)
				resp.Body.Close()
				if c.format == FormatSpringHealth {
					status, err := ensureSpringHealthFormat(b, resp)
					if err != nil {
						mainResult.Status = StatusDown
						mainResult.Message = err.Error()
					} else {
						mainResult.Status = status
					}
				}

				if c.contains != "" {
					if !strings.Contains(string(b), c.contains) {
						mainResult.Status = StatusDown
						mainResult.Message = fmt.Sprintf("missing string %q in result", c.contains)
					}
				}

				if mainResult.Status != StatusUp {
					mainResult.Detail = string(b)
				}
			}
		}
	}
	mainResult.Duration = int(time.Since(mainResult.Timestamp) / time.Millisecond)
	results := []Result{mainResult}

	return results
}

func ensureSpringHealthFormat(body []byte, resp *http.Response) (string, error) {
	if !strings.HasPrefix(resp.Header.Get("Content-Type"), "application/json") {
		return "DOWN", fmt.Errorf("expecting Content-Type: application/json, but got %v", resp.Header.Get("Content-Type"))
	}

	resultData := map[string]interface{}{}
	err := json.Unmarshal(body, &resultData)
	if err != nil {
		return "DOWN", errors.Wrap(err, "error parsing json body")
	}
	s, exist := resultData["status"]
	if !exist {
		return "DOWN", errors.New("missing status in response")
	}
	return fmt.Sprintf("%v", s), nil
}
