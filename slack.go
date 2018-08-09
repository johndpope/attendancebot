package main

import (
	"fmt"
	"strings"

	"encoding/json"
	"github.com/nlopes/slack"
	"io/ioutil"
	"os"
	"time"
	"unicode/utf8"
)

const (
	actionIn     = "in"
	actionOut    = "out"
	actionLeave  = "leave"
	actionCancel = "cancel"

	callbackID  = "punch"
	helpMessage = "```\nUsage:\tIntegration:\n\t\tauth\n\t\tadd [emp_id] [auth_code]\n\n\tDeintegration\t\tremove\n\nCheck In:\n\t\tin```"
)

type SlackListener struct {
	client    *slack.Client
	botID     string
	channelID string
}

func (s *SlackListener) ListenAndResponse() {
	rtm := s.client.NewRTM()
	go rtm.ManageConnection()

	for msg := range rtm.IncomingEvents {
		switch ev := msg.Data.(type) {
		case *slack.MessageEvent:
			if err := s.handleMessageEvent(ev); err != nil {
				sugar.Errorf("Failed to handle message: %s", err)
			}
		}
	}
}

func (s *SlackListener) handleMessageEvent(ev *slack.MessageEvent) error {
	if ev.Msg.SubType == "bot_message" {
		return nil
	}

	messageToBot := ev.Channel == s.channelID
	if messageToBot && ev.Msg.Text == "auth" {
		authURL := AuthCodeURL()
		return s.respond(ev.Channel, fmt.Sprintf("Please open the following URL in your browser:\n%s", authURL))
	}
	if messageToBot && (strings.HasPrefix(ev.Msg.Text, "register") || strings.HasPrefix(ev.Msg.Text, "add")) {
		split := strings.Fields(ev.Msg.Text)
		if len(split) != 3 {
			return s.respond(ev.Channel, "Invalid parameters.")
		}
		employeeID := split[1]
		code := split[2]
		if utf8.RuneCountInString(code) != 64 {
			return s.respond(ev.Channel, "Invalid authorization code.")
		}

		token, err := Token(code)

		user := User{
			ID:         ev.Msg.User,
			EmployeeID: employeeID,
			Token:      *token,
		}

		text, err := json.Marshal(user)
		if err != nil {
			return err
		}

		err = ioutil.WriteFile(fmt.Sprintf("users/%s", ev.Msg.User), text, 0644)
		if err != nil {
			return err
		}

		return s.respond(ev.Channel, ":ok: Saved your access token successfully.")
	}
	if messageToBot && (ev.Msg.Text == "unregister" || ev.Msg.Text == "remove") {
		err := os.Remove(fmt.Sprintf("users/%s", ev.Msg.User))
		if err != nil {
			s.respond(ev.Channel, fmt.Sprintf(":warning: Failed to remove '%s'.", ev.User))
			return err
		}

		return s.respond(ev.Channel, fmt.Sprintf(":ok: '%s' was removed successfully.", ev.User))
	}
	if messageToBot && (ev.Msg.Text == "punch" || ev.Msg.Text == "in" || ev.Msg.Text == "out" || ev.Msg.Text == "leave") {
		if _, _, err := s.client.PostMessage(ev.Msg.User, "", checkInOptions()); err != nil {
			return fmt.Errorf("failed to post message: %s", err)
		}
		return nil
	}
	if ev.Msg.Text == "ping" {
		return s.respond(ev.Channel, "pong")
	}
	if ev.Msg.Text == "help" {
		return s.respond(ev.Channel, helpMessage)
	}

	return nil
}

func (s *SlackListener) respond(channel string, text string) error {
	_, _, err := s.client.PostMessage(channel, text, slack.NewPostMessageParameters())
	return fmt.Errorf("failed to post message: %s", err)
}

func checkInOptions() slack.PostMessageParameters {
	attachment := slack.Attachment{
		Text:       time.Now().Format("2006/01/02 15:04"),
		CallbackID: callbackID,
		Actions: []slack.AttachmentAction{
			{
				Name: actionIn,
				Text: "Punch in",
				Type: "button",
			},
			{
				Name: actionOut,
				Text: "Punch out",
				Type: "button",
			},
			{
				Name:  actionLeave,
				Text:  "Leave",
				Type:  "button",
				Style: "danger",
			},
		},
	}
	parameters := slack.PostMessageParameters{
		Attachments: []slack.Attachment{
			attachment,
		},
	}
	return parameters
}

func (s *SlackListener) sendReminderMessage() error {
	ticker := time.NewTicker(40 * time.Minute)
	for {
		select {
		case <-ticker.C:
			location := time.FixedZone("Asia/Tokyo", 9*60*60)
			now := time.Now().In(location)
			if now.Hour() == 9 || now.Hour() == 17 {
				fileInfo, err := ioutil.ReadDir("users")
				if err != nil {
					return err
				}

				for _, file := range fileInfo {
					userID := file.Name()
					parameters := checkInOptions()

					if _, _, err := s.client.PostMessage(userID, "", parameters); err != nil {
						return fmt.Errorf("failed to post message: %s", err)
					}
				}
			}
		}
	}
	ticker.Stop()

	return nil
}
