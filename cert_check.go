package main

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"strconv"
	"time"

	"github.com/pkg/errors"
)

type CertCheck struct {
	environmentId string
	checkId       string
	name          string
	timeout       time.Duration
	host          string
	port          int
	minValidFor   time.Duration
}

func NewCertCheck(environmentId, checkId, name string, params map[string]string) (*CertCheck, error) {
	c := &CertCheck{
		environmentId: environmentId,
		checkId:       checkId,
		name:          name,
		host:          params["host"],
	}

	if p, exist := params["port"]; exist {
		port, err := strconv.Atoi(p)
		if err != nil {
			return nil, errors.Wrapf(err, "parsing port")
		}
		c.port = port
	} else {
		c.port = 443
	}

	if t, exist := params["timeout"]; exist {
		d, err := time.ParseDuration(t)
		if err != nil {
			return nil, errors.Wrapf(err, "parsing timeout")
		}
		c.timeout = d
	} else {
		c.timeout = time.Second * 10
	}

	if t, exist := params["minValidFor"]; exist {
		d, err := time.ParseDuration(t)
		if err != nil {
			return nil, errors.Wrapf(err, "parsing minValidFor")
		}
		c.minValidFor = d
	} else {
		c.minValidFor = 21 * 24 * time.Hour
	}

	return c, nil
}

func (c *CertCheck) Check() []Result {
	mainResult := NewResult(c.environmentId, c.checkId, c.name)

	mainResult.Status, mainResult.Message, mainResult.Detail = c.execute()

	mainResult.Duration = int(time.Since(mainResult.Timestamp) / time.Millisecond)

	results := []Result{mainResult}

	return results
}

func (c *CertCheck) execute() (status, message, detail string) {
	conn, err := tls.Dial("tcp", c.hostAndPort(), &tls.Config{})
	if err != nil {
		return StatusDown, err.Error(), ""
	}

	state := conn.ConnectionState()

	intermidiates := x509.NewCertPool()
	for _, c := range state.PeerCertificates {
		intermidiates.AddCert(c)
	}

	cert := state.PeerCertificates[0]
	_, err = cert.Verify(x509.VerifyOptions{
		DNSName:       c.host,
		Intermediates: intermidiates,
		CurrentTime:   time.Now().Add(c.minValidFor),
	})

	if err != nil {
		return StatusDown, err.Error(), ""
	}

	message = fmt.Sprintf("Valid from %v to %v", cert.NotBefore, cert.NotAfter)
	return StatusUp, message, ""
}

func (c *CertCheck) hostAndPort() string {
	return fmt.Sprintf("%s:%d", c.host, c.port)
}
