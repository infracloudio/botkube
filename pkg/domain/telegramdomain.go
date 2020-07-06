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

package domain

import (
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
)

// ITBot Listens to commands and message. Also listens to callBacks.
// This is designed to be immutable
type ITBot struct {
	token            string
	allowKubectl     bool
	restrictAccess   bool
	clusterName      string
	groupID          int64
	defaultNamespace string
	debug            bool
}

//WithConfig Used to create a new bot with a given config
func (bot ITBot) WithConfig(
	token string,
	allowKubectl bool,
	restrictAccess bool,
	clusterName string,
	groupID int64,
	defaultNamespace string,
	debug bool) ITBot {
	bot.token = token
	bot.allowKubectl = allowKubectl
	bot.restrictAccess = restrictAccess
	bot.clusterName = clusterName
	bot.groupID = groupID
	bot.defaultNamespace = defaultNamespace
	bot.debug = debug
	return bot
}

//Token telegram token
func (bot ITBot) Token() string {
	return bot.token
}

// AllowKubectl previlage
func (bot ITBot) AllowKubectl() bool {
	return bot.allowKubectl
}

// RestrictAccess previlage
func (bot ITBot) RestrictAccess() bool {
	return bot.restrictAccess
}

// ClusterName info
func (bot ITBot) ClusterName() string {
	return bot.clusterName
}

// GroupID of the telegram group. Usually is is a negetive value
func (bot ITBot) GroupID() int64 {
	return bot.groupID
}

//DefaultNamespace configuration
func (bot ITBot) DefaultNamespace() string {
	return bot.defaultNamespace
}

//Debug Enables the debug
func (bot ITBot) Debug() bool {
	return bot.debug
}

//ITMsg is an immutable message used to hold intermediate results of the pipeline
type ITMsg struct {
	request     tgbotapi.Update
	responseTxt string
	response    tgbotapi.MessageConfig
	command     string
}

//WithRequest create immutable telegram message with request
func (tMsg ITMsg) WithRequest(request tgbotapi.Update) ITMsg {
	tMsg.request = request
	return tMsg
}

//WithResponseTxt create immutable telegram message with response text
func (tMsg ITMsg) WithResponseTxt(responseTxt string) ITMsg {
	tMsg.responseTxt = responseTxt
	return tMsg
}

//WithResponse create immutable telegram message with Response
func (tMsg ITMsg) WithResponse(response tgbotapi.MessageConfig) ITMsg {
	tMsg.response = response
	return tMsg
}

//WithCommand create immutable telegram message with Command
func (tMsg ITMsg) WithCommand(command string) ITMsg {
	tMsg.command = command
	return tMsg
}

//IsLongResponse check size of the reponse
func (tMsg ITMsg) IsLongResponse() bool {
	if len(tMsg.responseTxt) >= 3990 {
		return true
	}
	return false
}

//HasKeyBoard check the response if it has a keyboard
func (tMsg ITMsg) HasKeyBoard() bool {
	if tMsg.response.ReplyMarkup != nil {
		return true
	}
	return false
}

//Request accessor
func (tMsg ITMsg) Request() tgbotapi.Update {
	return tMsg.request
}

//ResponseTxt accessor
func (tMsg ITMsg) ResponseTxt() string {
	return tMsg.responseTxt
}

//Response accessor
func (tMsg ITMsg) Response() tgbotapi.MessageConfig {
	return tMsg.response
}

//Command accessor
func (tMsg ITMsg) Command() string {
	return tMsg.command
}
