package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/pkg/errors"
)

type NotificationGateway struct {
	cfg *Config
}

func NewNotificationGateway(cfg *Config) *NotificationGateway {
	return &NotificationGateway{
		cfg: cfg,
	}
}

func (gw *NotificationGateway) NotifyDown(envId string, downtimes []*Downtime) error {
	var title string
	if len(downtimes) == 1 {
		title = fmt.Sprintf("[%v] CHECK DOWN: %v", envId, downtimes[0].Name)
	} else {
		title = fmt.Sprintf("[%v] %v CHECKS WENT DOWN", envId, len(downtimes))
	}

	body := bytes.NewBufferString("")
	for _, d := range downtimes {
		fmt.Fprintf(body, "%v (%v) is failing since %v\n--> %v\n", d.Name, d.Check, d.Start.Format("15:04:05 MST"), d.Message)
	}

	if gw.cfg.SelfUrl != "" {
		fmt.Fprintf(body, "See details at %v/#/%v\n", gw.cfg.SelfUrl, envId)
	}

	return gw.send(envId, title, body.String(), true)
}

func (gw *NotificationGateway) NotifyRecovered(envId string, downtimes []*Downtime) error {
	var title string
	if len(downtimes) == 1 {
		title = fmt.Sprintf("[%v] CHECK RECOVERED: %v", envId, downtimes[0].Name)
	} else {
		title = fmt.Sprintf("[%v] %v CHECKS RECOVERED", envId, len(downtimes))
	}

	body := bytes.NewBufferString("")
	for _, d := range downtimes {
		fmt.Fprintf(body, "%v (%v) recovered (was down for %v)\n", d.Name, d.Check, d.End.Sub(d.Start))
	}

	if gw.cfg.SelfUrl != "" {
		fmt.Fprintf(body, "See details at: %v/#/%v\n", gw.cfg.SelfUrl, envId)
	}

	return gw.send(envId, title, body.String(), false)
}

func (gw *NotificationGateway) send(envId, title, body string, isDown bool) error {
	log.Println(title + "\n" + body)
	notificationErrors := []string{}
	env, _ := gw.cfg.EnvById(envId)
	for _, n := range env.Notifications {
		isDaytime := isDaytime()
		alert := (n.AlertAtDaytime && isDaytime) || (n.AlertAtNighttime && !isDaytime)
		switch n.Type {
		case "hipchat":
			err := gw.sendHipchat(n.Target, title, body, isDown, alert)
			if err != nil {
				notificationErrors = append(notificationErrors, err.Error())
			}
		case "slack":
			err := gw.sendSlack(n.Target, title, body, isDown, alert)
			if err != nil {
				notificationErrors = append(notificationErrors, err.Error())
			}
		default:
			notificationErrors = append(notificationErrors, "notification type not supported: "+n.Type)
		}
	}
	if len(notificationErrors) > 0 {
		return errors.New(strings.Join(notificationErrors, ", "))
	}
	return nil
}

func (gw *NotificationGateway) sendHipchat(url, title, body string, isDown, alert bool) error {
	altertString := ""
	if alert {
		altertString = "@all "
	}
	msg := map[string]interface{}{
		"message":        altertString + title + "\n\n" + body,
		"message_format": "text",
		"notify":         true,
	}
	if isDown {
		msg["color"] = "red"
	} else {
		msg["color"] = "green"
	}

	b, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	resp, err := http.Post(url, "application/json", bytes.NewBuffer(b))
	if err != nil {
		return errors.Wrapf(err, "sending hipchat notification to %v", url)
	}
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return fmt.Errorf("got http status %v on sending hipchat notification to %v", resp.StatusCode, url)
	}
	return nil
}

func (gw *NotificationGateway) sendSlack(url, title, body string, isDown, alert bool) error {
	alertString := ""
	if alert {
		alertString = "@Channel "
	}

	color := "good"
	if isDown {
		color = "danger"
	}

	msg := map[string]interface{}{
		"attachments": []map[string]interface{}{
			map[string]interface{}{
				"fallback": alertString + title + "\n\n" + body,
				"color":    color,
				"title":    title,
				"text":     alertString + body,
			},
		},
	}

	b, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	resp, err := http.Post(url, "application/json", bytes.NewBuffer(b))
	if err != nil {
		return errors.Wrapf(err, "sending slack notification to %v", url)
	}
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return fmt.Errorf("got http status %v on sending slack notification to %v", resp.StatusCode, url)
	}
	return nil
}

func isDaytime() bool {
	hour := time.Now().Hour()
	return hour >= 7 && hour < 19
}
