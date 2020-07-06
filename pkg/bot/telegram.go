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
	"context"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
	"github.com/infracloudio/botkube/pkg/config"
	"github.com/infracloudio/botkube/pkg/domain"
	"github.com/infracloudio/botkube/pkg/execute"
	"github.com/infracloudio/botkube/pkg/log"
	"github.com/reactivex/rxgo/v2"
)

// TelegramBot Listens to commands and message. Also listens to callBacks.
// This is designed to be immutable
type telegramBot struct {
	iBot domain.ITBot
}

//NewTelegramBot will create a new telegram bot
func NewTelegramBot(c *config.Config) Bot {
	iBot := domain.ITBot{}.WithConfig(
		c.Communications.Telegram.Token,
		c.Settings.Kubectl.Enabled,
		c.Settings.Kubectl.RestrictAccess,
		c.Settings.ClusterName,
		c.Communications.Telegram.Groupid,
		c.Settings.Kubectl.DefaultNamespace,
		c.Communications.Telegram.Debug,
	)

	return &telegramBot{
		iBot: iBot,
	}
}

func isCommandF(itBot domain.ITBot, msg interface{}) bool {
	message := msg.(tgbotapi.Update)
	if message.Message.Chat.ID == itBot.GroupID() && message.Message.IsCommand() {
		return true
	}
	return false
}

func isCommandW(itBot domain.ITBot) func(interface{}) bool {
	return func(msg interface{}) bool {
		return isCommandF(itBot, msg)
	}
}

// processSimpleCommand is for handling simple commands like /help /config other commands which does not need kubectl
// creates new itMsg which will be used to hold intermediate results. By design it is immutable.
// itMsg structure should not be shared
func processSimpleCommand(ctx context.Context, msg interface{}) (interface{}, error) {
	message := msg.(tgbotapi.Update)
	reply := tgbotapi.NewMessage(message.Message.Chat.ID, "")
	reply.Text = "I recieved your command: " + message.Message.Text
	validCommand := "ignore"
	switch message.Message.Command() {
	case "help":
		reply.Text = "Avaliable commands /api_versions /ping \n /status /get  /logs"
	case "botkubehelp":
		reply.Text = "Avaliable commands /api_versions /ping \n /status /get  /logs"
	case "ping":
		validCommand = "ping"
	case "status":
		validCommand = "cluster-info"
	case "api_versions":
		validCommand = "api-versions"
	case "cluster_info":
		validCommand = "cluster-info"
	case "get":
		validCommand = "get " + message.Message.CommandArguments()
	case "logs":
		validCommand = "logs " + message.Message.CommandArguments()
	case "explain":
		validCommand = "explain " + message.Message.CommandArguments()
	case "top":
		validCommand = "top " + message.Message.CommandArguments()
	case "auth":
		validCommand = "auth " + message.Message.CommandArguments()
	default:
		reply.Text = "Command not supported. Please run /botkubehelp to see supported commands"
	}

	itMsg := domain.ITMsg{}.WithRequest(message).WithCommand(validCommand).WithResponse(reply)
	return itMsg, nil
}

func processKubeCommandF(ctx context.Context, msg interface{}, itBot domain.ITBot) (interface{}, error) {
	message := msg.(domain.ITMsg)
	commandExecutor := execute.NewDefaultExecutor(
		message.Command(),
		itBot.AllowKubectl(),
		itBot.RestrictAccess(),
		itBot.DefaultNamespace(),
		itBot.ClusterName(),
		" ",
		true)
	reply := message.Response()
	log.Debug("Inside processKubeCommandF   message.Command() != 'ignore'"+message.Command() != "ignore")
	if message.Command() != "ignore" {
		kubeResponse := commandExecutor.Execute()
		reply = tgbotapi.NewMessage(message.Request().Message.Chat.ID, "")
		reply.ReplyToMessageID = message.Request().Message.MessageID
		reply.Text = kubeResponse
	}

	itMsg := domain.ITMsg{}.WithRequest(message.Request()).WithCommand(message.Command()).WithResponse(reply)
	return itMsg, nil
}

func processKubeCommandW(itBot domain.ITBot) func(context.Context, interface{}) (interface{}, error) {
	return func(ctx context.Context, msg interface{}) (interface{}, error) {
		return processKubeCommandF(ctx, msg, itBot)
	}
}

// Start starts the telegrambot and  listens for messages
func (bot *telegramBot) Start() {
	isCommand := isCommandW(bot.iBot)
	processKubeCommand := processKubeCommandW(bot.iBot)
	tgbot, err := tgbotapi.NewBotAPI(bot.iBot.Token())
	if err != nil {
		log.Fatal("Error connecting to telegram check your token key. Are you using the latest? ", err)
	}
	tgbot.Debug = bot.iBot.Debug()

	updateConfig := tgbotapi.NewUpdate(0)
	updateConfig.Timeout = 60
	updatesChannel, err := tgbot.GetUpdatesChan(updateConfig)

	log.Info("BotKube connected to Telegram!")

	updatesObservable := rxgo.Just(updatesChannel,
		rxgo.WithBackPressureStrategy(rxgo.Drop),
		rxgo.WithPublishStrategy(),
		rxgo.WithPool(10),
		rxgo.WithErrorStrategy(rxgo.ContinueOnError))()
	pipiline := updatesObservable.Filter(isCommand).Map(processSimpleCommand).Map(processKubeCommand)
	replyChannel := make(chan rxgo.Item)
	pipiline.Send(replyChannel)

	for reply := range replyChannel {
		log.Info("Ready to reply :: ", reply.V)
		if reply.V.(domain.ITMsg).Response().Text != "" {
			if _, err := tgbot.Send(reply.V.(domain.ITMsg).Response()); err != nil {
				log.Fatal("Error occured :: ", err)
				log.Fatal(err)
			}
		}
	}

}
