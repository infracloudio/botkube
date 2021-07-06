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

package execute

import (
	"bytes"
	"fmt"
	"os"
	"reflect"
	"strings"
	"text/tabwriter"
	"unicode"

	"gopkg.in/yaml.v2"

	"github.com/infracloudio/botkube/pkg/config"
	filterengine "github.com/infracloudio/botkube/pkg/filterengine"
	"github.com/infracloudio/botkube/pkg/log"
	"github.com/infracloudio/botkube/pkg/utils"
)

const (
	defaultClusterCmd   = "cluster"
	defaultNamespaceCmd = "namespace"
)

var (
	//ValidKubectlConfigDefaultCommand set of commands used to define or query default values in a channel
	ValidKubectlConfigDefaultCommand = map[string]bool{
		defaultClusterCmd:   true,
		defaultNamespaceCmd: true,
	}
	// ValidNotifierCommand is a map of valid notifier commands
	ValidNotifierCommand = map[string]bool{
		"notifier": true,
	}
	validPingCommand = map[string]bool{
		"ping": true,
	}
	validVersionCommand = map[string]bool{
		"version": true,
	}
	validFilterCommand = map[string]bool{
		"filters": true,
	}
	validInfoCommand = map[string]bool{
		"commands": true,
	}
	validDebugCommands = map[string]bool{
		"exec":         true,
		"logs":         true,
		"attach":       true,
		"auth":         true,
		"api-versions": true,
		"cluster-info": true,
		"cordon":       true,
		"drain":        true,
		"uncordon":     true,
	}

	kubectlBinary = "/usr/local/bin/kubectl"
)

const (
	notifierStopMsg = "Sure! I won't send you notifications from cluster '%s' anymore."
	//UnsupportedCmdMsg message displayed when the command is not supported
	UnsupportedCmdMsg  = "Command not supported. Please run /botkubehelp to see supported commands."
	kubectlDisabledMsg = "Sorry, the admin hasn't given me the permission to execute kubectl command on cluster '%s'."
	filterNameMissing  = "You forgot to pass filter name. Please pass one of the following valid filters:\n\n%s"
	filterEnabled      = "I have enabled '%s' filter on '%s' cluster."
	filterDisabled     = "Done. I won't run '%s' filter on '%s' cluster."
	//DefaultClusterForKubectl message sent when querying default cluster for kubectl
	DefaultClusterForKubectl = "The default cluster for kubectl commands is : %s"
	//DefaultClusterForKubectlAccepted message sent when cluster is accepted to be default
	DefaultClusterForKubectlAccepted = "Using cluster %s as default for kubectl commands"

	//DefaultNamespaceForKubectl message sent when querying for default namespace
	DefaultNamespaceForKubectl = "The default namespace for cluster %s is %s"
	//DefaultNamespaceForKubectlAccepted message sent when default namespace is set
	DefaultNamespaceForKubectlAccepted = "Using default namespace %s for cluster %s"

	// NotifierStartMsg notifier enabled response message
	NotifierStartMsg = "Brace yourselves, notifications are coming from cluster '%s'."
	// IncompleteCmdMsg incomplete command response message
	IncompleteCmdMsg = "You missed to pass options for the command. Please run /botkubehelp to see command options."
	// WrongClusterCmdMsg incomplete command response message
	WrongClusterCmdMsg = "Sorry, the admin hasn't configured me to do that for the cluster '%s'."

	// Custom messages for teams platform
	teamsUnsupportedCmdMsg = "Command not supported. Please visit botkube.io/usage to see supported commands."
	teamsIncompleteCmdMsg  = "You missed to pass options for the command. Please run /botkubehelp to see command options."
)

// Executor is an interface for processes to execute commands
type Executor interface {
	Execute() string
}

// DefaultExecutor is a default implementations of Executor
type DefaultExecutor struct {
	Platform         config.BotPlatform
	Message          string
	AllowKubectl     bool
	RestrictAccess   bool
	ClusterName      string
	ChannelID        string
	IsAuthChannel    bool
	DefaultNamespace string
}

// CommandRunner is an interface to run bash commands
type CommandRunner interface {
	Run() (string, error)
}

// NotifierAction creates custom type for notifier actions
type NotifierAction string

// Defines constants for notifier actions
const (
	Start      NotifierAction = "start"
	Stop       NotifierAction = "stop"
	Status     NotifierAction = "status"
	ShowConfig NotifierAction = "showconfig"
)

func (action NotifierAction) String() string {
	return string(action)
}

// CommandFlags creates custom type for flags in botkube
type CommandFlags string

// Defines botkube flags
const (
	ClusterFlag    CommandFlags = "--cluster-name"
	FollowFlag     CommandFlags = "--follow"
	AbbrFollowFlag CommandFlags = "-f"
	WatchFlag      CommandFlags = "--watch"
	AbbrWatchFlag  CommandFlags = "-w"
)

func (flag CommandFlags) String() string {
	return string(flag)
}

// FiltersAction for options in filter commands
type FiltersAction string

// Filter command options
const (
	FilterList    FiltersAction = "list"
	FilterEnable  FiltersAction = "enable"
	FilterDisable FiltersAction = "disable"
)

// infoAction for options in Info commands
type infoAction string

// Info command options
const (
	infoList infoAction = "list"
)

func (action FiltersAction) String() string {
	return string(action)
}

// NewDefaultExecutor returns new Executor object
// msg should not contain the BotId
func NewDefaultExecutor(msg string, allowkubectl, restrictAccess bool, defaultNamespace,
	clusterName string, platform config.BotPlatform, channelID string, isAuthChannel bool) Executor {
	return &DefaultExecutor{
		Platform:         platform,
		Message:          msg,
		AllowKubectl:     allowkubectl,
		RestrictAccess:   restrictAccess,
		ClusterName:      clusterName,
		ChannelID:        channelID,
		IsAuthChannel:    isAuthChannel,
		DefaultNamespace: defaultNamespace,
	}
}

// Execute executes commands and returns output
func (e *DefaultExecutor) Execute() string {
	// Remove hyperlink if it got added automatically
	command := utils.RemoveHyperlink(e.Message)
	args := strings.Fields(strings.TrimSpace(command))
	if len(args) == 0 {
		if e.IsAuthChannel {
			return printDefaultMsg(e.Platform)
		}
		return "" // this prevents all bots on all clusters to answer something
	}
	if len(args) >= 1 && utils.AllowedKubectlVerbMap[args[0]] {
		if validDebugCommands[args[0]] || // Don't check for resource if is a valid debug command
			utils.AllowedKubectlResourceMap[args[1]] || // Check if allowed resource
			utils.AllowedKubectlResourceMap[utils.KindResourceMap[strings.ToLower(args[1])]] || // Check if matches with kind name
			utils.AllowedKubectlResourceMap[utils.ShortnameResourceMap[strings.ToLower(args[1])]] { // Check if matches with short name
			isClusterNamePresent := strings.Contains(e.Message, "--cluster-name")
			if !e.AllowKubectl { //kubectl not allowed so notify user
				_, isLinkedChannel := config.KubeCtlLinkedChannels[e.ChannelID]
				if (isClusterNamePresent && e.ClusterName == utils.GetClusterNameFromKubectlCmd(e.Message)) || isLinkedChannel {
					return fmt.Sprintf(kubectlDisabledMsg, e.ClusterName)
				}
				return ""
			}

			if e.RestrictAccess && !e.IsAuthChannel {
				return ""
			}
			return runKubectlCommand(args, e.ClusterName, e.DefaultNamespace, e.ChannelID)
		}
	}
	if ValidKubectlConfigDefaultCommand[args[0]] {
		return e.runDefaultCommand(args, e.ClusterName, e.ChannelID, e.AllowKubectl)
	}
	if ValidNotifierCommand[args[0]] {
		return e.runNotifierCommand(args, e.ClusterName, e.IsAuthChannel)
	}
	if validPingCommand[args[0]] {
		res := runVersionCommand(args, e.ClusterName)
		if len(res) == 0 {
			return ""
		}
		return fmt.Sprintf("pong from cluster '%s'", e.ClusterName) + "\n\n" + res
	}
	if validVersionCommand[args[0]] {
		return runVersionCommand(args, e.ClusterName)
	}
	// Check if filter command
	if validFilterCommand[args[0]] {
		return e.runFilterCommand(args, e.ClusterName, e.IsAuthChannel)
	}

	//Check if info command
	if validInfoCommand[args[0]] {
		return e.runInfoCommand(args, e.IsAuthChannel)
	}

	if e.IsAuthChannel {
		return printDefaultMsg(e.Platform)
	}
	return ""
}

func printDefaultMsg(p config.BotPlatform) string {
	if p == config.TeamsBot {
		return teamsUnsupportedCmdMsg
	}
	return UnsupportedCmdMsg
}

// Trim single and double quotes from ends of string
func trimQuotes(clusterValue string) string {
	return strings.TrimFunc(clusterValue, func(r rune) bool {
		if r == unicode.SimpleFold('\u0027') || r == unicode.SimpleFold('\u0022') {
			return true
		}
		return false
	})
}

func runKubectlCommand(args []string, clusterName, defaultNamespace string, channelID string) string {
	_, isLinkedChannel := config.KubeCtlLinkedChannels[channelID]
	if isLinkedChannel { //use default channel set by user
		defaultNamespace = config.KubeCtlLinkedChannels[channelID]
	}
	// run commands in namespace specified under Config.Settings.DefaultNamespace field
	if !utils.Contains(args, "-n") && !utils.Contains(args, "--namespace") && len(defaultNamespace) != 0 {
		args = append([]string{"-n", defaultNamespace}, utils.DeleteDoubleWhiteSpace(args)...)
	}

	// Remove unnecessary flags
	finalArgs := []string{}
	isClusterNameArg := false
	isTargetingThisChannel := false
	for index, arg := range args {
		if isClusterNameArg {
			isClusterNameArg = false
			continue
		}
		if arg == AbbrFollowFlag.String() || strings.HasPrefix(arg, FollowFlag.String()) {
			continue
		}
		if arg == AbbrWatchFlag.String() || strings.HasPrefix(arg, WatchFlag.String()) {
			continue
		}
		// Check --cluster-name flag
		if strings.HasPrefix(arg, ClusterFlag.String()) {
			// Check if flag value in current or next argument and compare with config.settings.clustername
			if arg == ClusterFlag.String() {
				if index == len(args)-1 || trimQuotes(args[index+1]) != clusterName {
					return ""
				}
				isClusterNameArg = true
			} else {
				if trimQuotes(strings.SplitAfterN(arg, ClusterFlag.String()+"=", 2)[1]) != clusterName {
					return ""
				}
			}
			isTargetingThisChannel = true
			continue
		}
		finalArgs = append(finalArgs, arg)
	}
	if isTargetingThisChannel == false && !isLinkedChannel {
		return ""
	}
	// Get command runner
	runner := NewCommandRunner(kubectlBinary, finalArgs)
	out, err := runner.Run()
	if err != nil {
		log.Error("Error in executing kubectl command: ", err)
		return fmt.Sprintf("Cluster: %s\n%s", clusterName, out+err.Error())
	}
	return fmt.Sprintf("Cluster: %s\n%s", clusterName, out)
}

// TODO: Have a separate cli which runs bot commands

//runDefaultCommand execute kubectl config default commands
//if the command has not arguments then return the default value
//if the command has arguments then use those to set the default value.
func (e *DefaultExecutor) runDefaultCommand(args []string, clusterName string, channelID string, kubectlAllowed bool) string {
	_, isManagedChannel := config.KubeCtlLinkedChannels[channelID]
	//if one 1 arg then this queries the default cluster value
	if len(args) == 1 {
		switch args[0] {
		case defaultClusterCmd:
			if isManagedChannel {
				return fmt.Sprintf(DefaultClusterForKubectl, clusterName)
			} // else another cluster may be the default
			return ""
		case defaultNamespaceCmd:
			if isManagedChannel {
				return fmt.Sprintf(DefaultNamespaceForKubectl, clusterName, config.KubeCtlLinkedChannels[channelID])
			} // else another cluster may be the default
			return ""
		default: //unhandled command
			return ""
		}
	}
	//if more that 1 arg then check if this bot cluster is the requested default, then record it
	if len(args) > 1 {
		switch args[0] {
		case defaultClusterCmd:
			if args[1] == clusterName { //default command is targeting this cluster :)
				if kubectlAllowed {
					//adding the channel to the channels linked to this cluster
					namespace := config.KubeCtlLinkedChannels[channelID] //to avoid changing the namespace is it was previously set
					if namespace == "" {
						namespace = "default"
					}
					config.KubeCtlLinkedChannels[channelID] = namespace
					return fmt.Sprintf(DefaultClusterForKubectlAccepted, clusterName)
				} //else kubectl not allowed on this channel so say it
				return fmt.Sprintf(kubectlDisabledMsg, clusterName)
			} //else removes this channel from the list of channel linked to this cluster for kubectl
			//we don't care if it was already in the set or not
			delete(config.KubeCtlLinkedChannels, channelID)
		case defaultNamespaceCmd: //want to set default namespace so check if it is the right cluster
			if isManagedChannel {
				config.KubeCtlLinkedChannels[channelID] = args[1]
				return fmt.Sprintf(DefaultNamespaceForKubectlAccepted, args[1], clusterName)
			} // else another cluster may be the default
			return ""
		default: //unhandled command
			return ""
		}
	}
	return ""
}

func (e *DefaultExecutor) runNotifierCommand(args []string, clusterName string, isAuthChannel bool) string {
	if isAuthChannel == false {
		return ""
	}
	if len(args) < 2 {
		return IncompleteCmdMsg
	}

	switch args[1] {
	case Start.String():
		config.Notify = true
		log.Info("Notifier enabled")
		return fmt.Sprintf(NotifierStartMsg, clusterName)
	case Stop.String():
		config.Notify = false
		log.Info("Notifier disabled")
		return fmt.Sprintf(notifierStopMsg, clusterName)
	case Status.String():
		if config.Notify == false {
			return fmt.Sprintf("Notifications are off for cluster '%s'", clusterName)
		}
		return fmt.Sprintf("Notifications are on for cluster '%s'", clusterName)
	case ShowConfig.String():
		out, err := showControllerConfig()
		if err != nil {
			log.Error("Error in executing showconfig command: ", err)
			return "Error in getting configuration!"
		}
		return fmt.Sprintf("Showing config for cluster '%s'\n\n%s", clusterName, out)
	}
	return printDefaultMsg(e.Platform)
}

// runFilterCommand to list, enable or disable filters
func (e *DefaultExecutor) runFilterCommand(args []string, clusterName string, isAuthChannel bool) string {
	if isAuthChannel == false {
		return ""
	}
	if len(args) < 2 {
		return IncompleteCmdMsg
	}

	switch args[1] {
	case FilterList.String():
		log.Debug("List filters")
		return makeFiltersList()

	// Enable filter
	case FilterEnable.String():
		if len(args) < 3 {
			return fmt.Sprintf(filterNameMissing, makeFiltersList())
		}
		log.Debug("Enable filters", args[2])
		if err := filterengine.DefaultFilterEngine.SetFilter(args[2], true); err != nil {
			return err.Error()
		}
		return fmt.Sprintf(filterEnabled, args[2], clusterName)

	// Disable filter
	case FilterDisable.String():
		if len(args) < 3 {
			return fmt.Sprintf(filterNameMissing, makeFiltersList())
		}
		log.Debug("Disabled filters", args[2])
		if err := filterengine.DefaultFilterEngine.SetFilter(args[2], false); err != nil {
			return err.Error()
		}
		return fmt.Sprintf(filterDisabled, args[2], clusterName)
	}
	return printDefaultMsg(e.Platform)
}

//runInfoCommand to list allowed commands
func (e *DefaultExecutor) runInfoCommand(args []string, isAuthChannel bool) string {
	if isAuthChannel == false {
		return ""
	}
	if len(args) < 2 && args[1] != string(infoList) {
		return IncompleteCmdMsg
	}

	if len(args) > 3 && args[2] == ClusterFlag.String() && args[3] != e.ClusterName {
		return fmt.Sprintf(WrongClusterCmdMsg, args[3])
	}

	return makeCommandInfoList()
}

func makeCommandInfoList() string {
	allowedVerbs := utils.GetStringInYamlFormat("allowed verbs:", utils.AllowedKubectlVerbMap)
	allowedResources := utils.GetStringInYamlFormat("allowed resources:", utils.AllowedKubectlResourceMap)
	return allowedVerbs + allowedResources
}

// Use tabwriter to display string in tabular form
// https://golang.org/pkg/text/tabwriter
func makeFiltersList() string {
	buf := new(bytes.Buffer)
	w := tabwriter.NewWriter(buf, 5, 0, 1, ' ', 0)

	fmt.Fprintln(w, "FILTER\tENABLED\tDESCRIPTION")
	for k, v := range filterengine.DefaultFilterEngine.ShowFilters() {
		fmt.Fprintf(w, "%s\t%v\t%s\n", reflect.TypeOf(k).Name(), v, k.Describe())
	}

	w.Flush()
	return buf.String()
}

func findBotKubeVersion() (versions string) {
	args := []string{"-c", fmt.Sprintf("%s version --short=true | grep Server", kubectlBinary)}
	runner := NewCommandRunner("sh", args)
	// Returns "Server Version: xxxx"
	k8sVersion, err := runner.Run()
	if err != nil {
		log.Warn(fmt.Sprintf("Failed to get Kubernetes version: %s", err.Error()))
		k8sVersion = "Server Version: Unknown\n"
	}

	botkubeVersion := os.Getenv("BOTKUBE_VERSION")
	if len(botkubeVersion) == 0 {
		botkubeVersion = "Unknown"
	}
	return fmt.Sprintf("K8s %sBotKube version: %s", k8sVersion, botkubeVersion)
}

func runVersionCommand(args []string, clusterName string) string {
	checkFlag := false
	for _, arg := range args {
		if checkFlag {
			if arg != clusterName {
				return ""
			}
			checkFlag = false
			continue
		}
		if strings.HasPrefix(arg, ClusterFlag.String()) {
			if arg == ClusterFlag.String() {
				checkFlag = true
			} else if strings.SplitAfterN(arg, ClusterFlag.String()+"=", 2)[1] != clusterName {
				return ""
			}
			continue
		}
	}
	return findBotKubeVersion()
}

func showControllerConfig() (configYaml string, err error) {
	c, err := config.New()
	if err != nil {
		log.Fatal(fmt.Sprintf("Error in loading configuration. Error:%s", err.Error()))
	}

	// hide sensitive info
	c.Communications.Slack.Token = ""
	c.Communications.ElasticSearch.Password = ""

	b, err := yaml.Marshal(c)
	if err != nil {
		return configYaml, err
	}
	configYaml = string(b)

	return configYaml, nil
}
