// Copyright (c) 2019 InfraCloud Technologies
//
// Permission is hereby granted, free of charge, to any person obtaining a copy of
// this software and associated documentation files (the "Software"), to deal in
// the Software without restriction, including without limitation the rights to
// use, copy, modify, merge, publish, distribute, sublicense, and/or sell copies of
// the Software, and to permit persons to whom the Software is furnished to do so,
// subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY, FITNESS
// FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR
// COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER
// IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN
// CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.

package bot

import (
	"fmt"
	"strings"
	"testing"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
	"github.com/infracloudio/botkube/pkg/domain"
)

func TestIsCommand(t *testing.T) {

	iBot := domain.ITBot{}.WithConfig(
		"SOME_TOKEN",
		true,
		true,
		"microk8s",
		-100,
		"default",
		true,
	)

	message := tgbotapi.Message{Text: "/command"}
	chat := tgbotapi.Chat{ID: -100, Type: "group"}
	message.Entities = &[]tgbotapi.MessageEntity{{Type: "bot_command", Offset: 0, Length: 8}}
	message.Chat = &chat
	update := tgbotapi.Update{
		UpdateID: 200,
		Message:  &message,
	}
	result := isCommandF(iBot, update)
	if !result {
		t.Error("Abs(-1)", result)
	}
}

func TestIsCommandDataDriven(t *testing.T) {
	var testData = []struct {
		messageText string
		chatid      int64
		expected    bool
	}{
		{"command", -100, false},
		{"help", -100, false},
	}
	iBot := domain.ITBot{}.WithConfig(
		"SOME_TOKEN",
		true,
		true,
		"microk8s",
		-100,
		"default",
		true,
	)
	for _, td := range testData {
		testname := fmt.Sprintf("%s,%d", td.messageText, td.chatid)

		message := tgbotapi.Message{Text: td.messageText}
		chat := tgbotapi.Chat{ID: td.chatid, Type: "group"}
		message.Chat = &chat
		update := tgbotapi.Update{
			UpdateID: 200,
			Message:  &message,
		}
		t.Run(testname, func(t *testing.T) {
			result := isCommandF(iBot, update)
			// fmt.Printf("\n got %t, expected  %t messageText %s groupid %d ", result, td.expected, td.messageText, td.chatid)
			if result != td.expected {
				fmt.Printf("Failed got %t, expected  %t messageText %s groupid %d ", result, td.expected, td.messageText, td.chatid)
				t.Errorf("got %t, expected  %t messageText %s groupid %d ", result, td.expected, td.messageText, td.chatid)

			}

		})
	}
}

func TestIsCommandWithCommandDataDriven(t *testing.T) {
	var testData = []struct {
		messageText string
		chatid      int64
		expected    bool
	}{
		{"/command", -100, true},
		{"/help", -100, true},
		{"/help", -101, false},
		{"/get", -100, true},
		{"/get pods", -100, true},
		{"/api_resources", -100, true},
	}
	iBot := domain.ITBot{}.WithConfig(
		"SOME_TOKEN",
		true,
		true,
		"microk8s",
		-100,
		"default",
		true,
	)
	for _, td := range testData {
		testname := fmt.Sprintf("%s,%d", td.messageText, td.chatid)

		message := tgbotapi.Message{Text: td.messageText}
		chat := tgbotapi.Chat{ID: td.chatid, Type: "group"}
		message.Entities = &[]tgbotapi.MessageEntity{{Type: "bot_command", Offset: 0, Length: len(td.messageText)}}
		message.Chat = &chat
		update := tgbotapi.Update{
			UpdateID: 200,
			Message:  &message,
		}
		t.Run(testname, func(t *testing.T) {
			result := isCommandF(iBot, update)
			if result != td.expected {
				fmt.Printf("Failed got %t, expected  %t messageText %s groupid %d ", result, td.expected, td.messageText, td.chatid)
				t.Errorf("got %t, expected  %t messageText %s groupid %d ", result, td.expected, td.messageText, td.chatid)
			}

		})
	}
}

func TestProcessSimpleCommand(t *testing.T) {

	var testData = []struct {
		command         string
		expectedReply   string
		expectedCommand string
	}{
		{"/get pods", "I recieved your command: /get pods", "get pods"},
		{"/help", "Avaliable commands /api_versions /ping \n /status /get  /logs", "ignore"},
		{"/botkubehelp", "Avaliable commands /api_versions /ping \n /status /get  /logs", "ignore"},
		{"/junk", "Command not supported. Please run /botkubehelp to see supported commands", "ignore"},
		{"/ping", "I recieved your command: /ping", "ping"},
		{"/status", "I recieved your command: /status", "cluster-info"},
		{"/api_versions", "I recieved your command: /api_versions", "api-versions"},
		{"/cluster_info", "I recieved your command: /cluster_info", "cluster-info"},
		{"/get", "I recieved your command: /get", "get "},
		// {"/get pods ", "I recieved your command: /get pods", "get pods"},
		{"/logs name ", "I recieved your command: /logs name ", "logs name "},
		{"/logs ", "I recieved your command: /logs ", "logs "},
		{"/explain ", "I recieved your command: /explain ", "explain "},
		{"/explain name ", "I recieved your command: /explain name ", "explain name "},
		{"/top name ", "I recieved your command: /top name ", "top name "},
		{"/auth name ", "I recieved your command: /auth name ", "auth name "},
	}

	for _, td := range testData {
		testname := fmt.Sprintf("%s,%s", td.command, td.expectedCommand)

		message := tgbotapi.Message{Text: td.command}
		chat := tgbotapi.Chat{ID: -100, Type: "group"}
		l := strings.Index(td.command, " ")
		commandLen := l
		if l < 0 {
			commandLen = len(td.command)
		}
		message.Entities = &[]tgbotapi.MessageEntity{{Type: "bot_command", Offset: 0, Length: commandLen}}
		message.Chat = &chat
		update := tgbotapi.Update{
			UpdateID: 200,
			Message:  &message,
		}
		// cmd := message.Command()
		// fmt.Printf(" cmd %v ", cmd)
		t.Run(testname, func(t *testing.T) {
			response, err := processSimpleCommand(nil, update)
			if err != nil {
				t.Errorf("Error occured while processing command %v Test %v ", td.command, testname)
			}
			processingResult := response.(domain.ITMsg)
			if processingResult.Command() != td.expectedCommand {
				t.Errorf("got %q, expected command %q ResponseTxt %q ",
					processingResult.Command(), td.expectedCommand, processingResult.ResponseTxt())
			}
			if processingResult.Response().Text != td.expectedReply {
				t.Errorf("got %q, expected ResponseTxt  %q command %q ",
					processingResult.Response().Text, td.expectedReply, processingResult.Command())

			}
		})

	}

}
