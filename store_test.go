package main

import (
	"fmt"
	. "github.com/stretchr/testify/assert"
	"io/ioutil"
	"os/exec"
	"testing"
	"time"
)

func Test_Store_CheckStatus(t *testing.T) {
	cfg := testConfig(t)
	store, err := NewStore(cfg, NewNotificationGateway(cfg))
	NoError(t, err)
	defer store.Close()

	up := upResult("check1")
	NoError(t, store.InsertResult(up))

	s, err := store.Status("testEnv")
	NoError(t, err)

	Equal(t, 2, len(s))
	Equal(t, "check1", s[0].Check)
	Equal(t, "testEnv", s[0].Environment)
	Equal(t, "", s[0].Message)
	Equal(t, "Check 1", s[0].Name)
	Equal(t, "UP", s[0].Status)
	Equal(t, 42, s[0].Duration)
	Equal(t, up.Id, s[0].LastResultId)
	InDelta(t, s[0].Updated.Unix(), time.Now().Unix(), 1000)

	down := downResult("check1")
	NoError(t, store.InsertResult(down))

	s, err = store.Status("testEnv")
	NoError(t, err)

	Equal(t, 2, len(s))
	Equal(t, "check1", down.Check)
	Equal(t, down.Message, s[0].Message)
	Equal(t, down.Status, s[0].Status)
	Equal(t, down.Duration, s[0].Duration)
	Equal(t, down.Id, s[0].LastResultId)
	InDelta(t, s[0].Updated.Unix(), time.Now().Unix(), 1000)
}

func Test_Store_CheckStatus_UpdateNames(t *testing.T) {
	cfg := testConfig(t)
	store, err := NewStore(cfg, NewNotificationGateway(cfg))
	NoError(t, err)
	defer store.Close()

	checkId := cfg.Environments[0].Checks[0].Id
	cfg.Environments[0].Checks[0].Name = "updated name"
	store, err = NewStore(cfg, NewNotificationGateway(cfg))
	NoError(t, err)
	defer store.Close()

	s, err := store.Status(cfg.Environments[0].Id)
	NoError(t, err)

	var changedStatus *CheckStatus
	for _, res := range s {
		if res.Check == checkId {
			changedStatus = res
		}
	}

	Equal(t, "updated name", changedStatus.Name)
}

func Test_Store_DowntimeNotifications(t *testing.T) {
	cfg := testConfig(t)
	notifyMock := &NotifyMock{}
	store, err := NewStore(cfg, notifyMock)
	NoError(t, err)
	defer store.Close()

	NoError(t, store.InsertResult(upResult("check1")))
	notifyMock.AssertNoNotifications(t)

	NoError(t, store.InsertResult(downResult("check1")))
	notifyMock.AssertNoNotifications(t)

	NoError(t, store.InsertResult(downResult("check1")))
	Equal(t, 1, len(notifyMock.downs))
	Equal(t, "testEnv", notifyMock.environment)
	Nil(t, notifyMock.ups)
	notifyMock.reset()

	NoError(t, store.InsertResult(upResult("check1")))
	Equal(t, 1, len(notifyMock.ups))
	Nil(t, notifyMock.downs)

	NoError(t, store.InsertResult(downResult("check1")))

	NoError(t, store.InsertResult(downResult("check2")))
	NoError(t, store.InsertResult(upResult("check2")))

	downtimes, err := store.Downtimes("testEnv")
	NoError(t, err)

	// first downtime is the one, which is not recovered
	d := downtimes[0]
	Equal(t, "check1", d.Check)
	Equal(t, 1, d.FailCount)
	False(t, d.Recovered)

	// second downtime is the newest one
	d = downtimes[1]
	Equal(t, "check2", d.Check)

	// 3rd downtime is the oldest one
	d = downtimes[2]
	Equal(t, 3, len(downtimes))
	Equal(t, "check1", d.Check)
	Equal(t, "check1", d.Name)
	Equal(t, 2, d.FailCount)
	True(t, d.Recovered)
}

func dumpDB(file string) {
	out, err := exec.Command("sqlite3", file, ".dump").Output()
	if err != nil {
		panic(err)
	}
	fmt.Printf("DB output %v:\n%s\n", file, out)
}

var testSequence uint = 0

func upResult(check string) Result {
	testSequence++
	return Result{
		Id:          uint(testSequence),
		Environment: "testEnv",
		Check:       check,
		Name:        check,
		Status:      "UP",
		Message:     "",
		Detail:      "",
		Duration:    42,
		Timestamp:   time.Now(),
	}
}

func downResult(check string) Result {
	testSequence++
	return Result{
		Id:          testSequence,
		Environment: "testEnv",
		Check:       check,
		Name:        check,
		Status:      "DOWN",
		Message:     "some error",
		Detail:      "some error detail",
		Duration:    42,
		Timestamp:   time.Now(),
	}
}

func testConfig(t *testing.T) *Config {
	f, err := ioutil.TempFile("", "insantus_unittest")
	NoError(t, err)
	f.Close()
	return &Config{
		DBPath: f.Name(),
		Environments: []Env{
			Env{
				Id:            "testEnv",
				Name:          "testEnv",
				Notifications: []Notification{},
				Checks: []Check{
					Check{
						Id:   "check1",
						Name: "Check 1",
						Type: "http",
					},
					Check{
						Id:   "check2",
						Name: "Check 2",
						Type: "http",
					},
				},
			},
		},
	}
}

type NotifyMock struct {
	environment string
	downs       []*Downtime
	ups         []*Downtime
}

func (nm *NotifyMock) reset() {
	nm.environment = ""
	nm.downs = nil
	nm.ups = nil
}

func (nm *NotifyMock) NotifyDown(envId string, downtimes []*Downtime) error {
	nm.environment = envId
	nm.downs = downtimes
	return nil
}
func (nm *NotifyMock) NotifyRecovered(envId string, downtimes []*Downtime) error {
	nm.environment = envId
	nm.ups = downtimes
	return nil
}

func (nm *NotifyMock) AssertNoNotifications(t *testing.T) {
	Equal(t, "", nm.environment)
	Nil(t, nm.downs)
	Nil(t, nm.ups)
}
