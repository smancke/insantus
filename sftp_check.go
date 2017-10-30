package main

import (
	"fmt"
	"github.com/pkg/errors"
	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
	"time"
)

type SftpCheck struct {
	environmentId string
	checkId       string
	name          string

	timeout  time.Duration
	host     string
	port     string
	user     string
	password string
	key      string
	hostKey  string
	testfile string
}

func NewSftpCheck(environmentId, checkId, name string, params map[string]string) (*SftpCheck, error) {
	c := &SftpCheck{
		environmentId: environmentId,
		checkId:       checkId,
		name:          name,
		host:          params["host"],
		port:          params["port"],
		user:          params["user"],
		password:      params["password"],
		key:           params["key"],
		hostKey:       params["hostKey"],
		testfile:      params["testfile"],
	}

	if c.password == "" && c.key == "" {
		return nil, fmt.Errorf("password or key have to be supplied")
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

	if c.port == "" {
		c.port = "22"
	}

	return c, nil
}

func (c *SftpCheck) Check() []Result {
	mainResult := NewResult(c.environmentId, c.checkId, c.name)

	var err error
	done := make(chan bool, 1)
	go func() {
		err = c.connectTest()
		done <- true
	}()

	select {
	case <-done:
	case <-time.After(c.timeout):
		err = fmt.Errorf("sftp timeout after %v on %v:%v", c.timeout, c.host, c.port)
	}

	if err != nil {
		mainResult.Status = StatusDown
		mainResult.Message = err.Error()
	} else {
		mainResult.Status = StatusUp
	}
	mainResult.Duration = int(time.Since(mainResult.Timestamp) / time.Millisecond)
	results := []Result{mainResult}

	return results
}

func (c *SftpCheck) connectTest() error {

	var authMethod ssh.AuthMethod
	if c.key != "" {
		signer, err := ssh.ParsePrivateKey([]byte(c.key))
		if err != nil {
			return errors.Wrap(err, "unable to parse private key")
		}
		authMethod = ssh.PublicKeys(signer)
	} else {
		authMethod = ssh.Password(c.password)
	}

	var hostKeyVerification ssh.HostKeyCallback
	if c.hostKey != "" {
		hostKey, _, _, _, err := ssh.ParseAuthorizedKey([]byte(c.hostKey))
		if err != nil {
			return errors.Wrap(err, "unable to parse host key")
		}
		hostKeyVerification = ssh.FixedHostKey(hostKey)
	} else {
		hostKeyVerification = ssh.InsecureIgnoreHostKey()
	}

	config := &ssh.ClientConfig{
		User:            c.user,
		Auth:            []ssh.AuthMethod{authMethod},
		HostKeyCallback: hostKeyVerification,
	}
	conn, err := ssh.Dial("tcp", c.host+":"+c.port, config)
	if err != nil {
		return errors.Wrap(err, "failed to connect with ssh")
	}
	defer conn.Close()

	sftp, err := sftp.NewClient(conn)
	if err != nil {
		return errors.Wrap(err, "can't create sftp client")
	}
	defer sftp.Close()

	// Check if we can create a file
	if c.testfile != "" {
		f, err := sftp.Create(c.testfile)
		if err != nil {
			return errors.Wrap(err, "can not create testfile")
		}
		if _, err := f.Write([]byte("Healthcheck")); err != nil {
			return errors.Wrap(err, "can not write to testfile")
		}
		f.Close()

		_, err = sftp.Lstat(c.testfile)
		if err != nil {
			return errors.Wrap(err, "testfile not there")
		}

		err = sftp.Remove(c.testfile)
		if err != nil {
			return errors.Wrap(err, "can not remove testfile")
		}
	}

	return nil
}
