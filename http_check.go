package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"time"
)

type HttpCheck struct {
	environmentId string
	checkId       string
	name          string
	timeout       time.Duration
	url           string
	user          string
	password      string
}

func NewHttpCheck(environmentId, checkId, name string, params map[string]string) (*HttpCheck, error) {
	c := &HttpCheck{
		environmentId: environmentId,
		checkId:       checkId,
		name:          name,
		url:           params["url"],
		user:          params["user"],
		password:      params["password"],
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

			if resp.Body != nil {
				b, _ := ioutil.ReadAll(resp.Body)
				resp.Body.Close()
				if strings.HasPrefix(resp.Header.Get("Content-Type"), "application/json") {
					resultData := map[string]interface{}{}
					err := json.Unmarshal(b, &resultData)
					if err != nil {
						mainResult.Status = StatusDown
						mainResult.Message = mainResult.Message + fmt.Sprintf("error parsing json body: %v\n", err)
					} else {
						if s, exist := resultData["status"]; exist {
							mainResult.Status = fmt.Sprintf("%v", s)
						}
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
