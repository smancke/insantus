package main

import (
	"fmt"
	"sync"
	"time"

	"github.com/pkg/errors"
	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
)

type SftpCheck struct {
	environmentID string
	checkID       string
	name          string

	timeout      time.Duration
	host         string
	port         string
	user         string
	password     string
	key          string
	hostKey      string
	testfile     string
	clientConfig *ssh.ClientConfig

	mutex sync.Mutex
}

func NewSftpCheck(environmentID, checkID, name string, params map[string]string) (*SftpCheck, error) {
	c := &SftpCheck{
		environmentID: environmentID,
		checkID:       checkID,
		name:          name,
		host:          params["host"],
		port:          params["port"],
		testfile:      params["testfile"],
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

	clientConfig, err := createSSHClientConfig(
		params["user"],
		params["key"],
		params["password"],
		params["hostKey"],
	)
	if err != nil {
		return nil, err
	}

	c.clientConfig = clientConfig

	return c, nil
}

func (c *SftpCheck) Check() []Result {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	mainResult := NewResult(c.environmentID, c.checkID, c.name)

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
	conn, err := ssh.Dial("tcp", c.host+":"+c.port, c.clientConfig)
	if err != nil {
		return errors.Wrap(err, "failed to connect with ssh")
	}
	defer conn.Close()

	client, err := sftp.NewClient(conn)
	if err != nil {
		return errors.Wrap(err, "can't create sftp client")
	}
	defer client.Close()

	// Check if we can create a file
	if c.testfile != "" {
		return c.checkFileCreation(client)
	}

	return nil
}

func (c *SftpCheck) checkFileCreation(client *sftp.Client) error {
	f, err := client.Create(c.testfile)
	if err != nil {
		return errors.Wrap(err, "can not create testfile")
	}
	if _, err := f.Write([]byte("Healthcheck")); err != nil {
		return errors.Wrap(err, "can not write to testfile")
	}
	f.Close()

	_, err = client.Lstat(c.testfile)
	if err != nil {
		return errors.Wrap(err, "testfile not there")
	}

	err = client.Remove(c.testfile)
	if err != nil {
		return errors.Wrap(err, "can not remove testfile")
	}
	return nil
}

func createSSHClientConfig(user, key, password, hostKey string) (*ssh.ClientConfig, error) {
	if password == "" && key == "" {
		return nil, fmt.Errorf("password or key have to be supplied")
	}

	authMethod, err := selectAuthMethod(key, password)
	if err != nil {
		return nil, err
	}
	hostKeyVerification, err := selectHostKeyVerification(hostKey)
	if err != nil {
		return nil, err
	}

	return &ssh.ClientConfig{
		User:            user,
		Auth:            []ssh.AuthMethod{authMethod},
		HostKeyCallback: hostKeyVerification,
	}, nil
}

func selectAuthMethod(key, password string) (ssh.AuthMethod, error) {
	if key != "" {
		signer, err := ssh.ParsePrivateKey([]byte(key))
		if err != nil {
			return nil, errors.Wrap(err, "unable to parse private key")
		}
		return ssh.PublicKeys(signer), nil
	}

	return ssh.Password(password), nil
}

func selectHostKeyVerification(hostKey string) (ssh.HostKeyCallback, error) {
	if hostKey != "" {
		hostKey, _, _, _, err := ssh.ParseAuthorizedKey([]byte(hostKey))
		if err != nil {
			return nil, errors.Wrap(err, "unable to parse host key")
		}
		return ssh.FixedHostKey(hostKey), nil
	}

	return ssh.InsecureIgnoreHostKey(), nil
}
