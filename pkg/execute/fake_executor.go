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

// +build test

package execute

import (
	"fmt"
	"strings"
)

// K8sVersion fake version send in ping response
const K8sVersion = "v1.15.3"

// KubectlResponse map for fake Kubectl responses
var KubectlResponse = map[string]string{
	"-n default get pods": "NAME                           READY   STATUS    RESTARTS   AGE\n" +
		"nginx-xxxxxxx-yyyyyyy          1/1     Running   1          1d",
	"-c " + kubectlBinary + " version --short=true | grep Server": fmt.Sprintf("Server Version: %s\n", K8sVersion),
	"-n foo get pods": "result for get pods",
}

// FakeRunner mocks Run
type FakeRunner struct {
	command string
	args    []string
}

// NewCommandRunner returns new DefaultRunner
func NewCommandRunner(command string, args []string) CommandRunner {
	return FakeRunner{
		command: command,
		args:    args,
	}
}

// Run executes bash command
func (r FakeRunner) Run() (string, error) {
	cmd := strings.Join(r.args, " ")
	return KubectlResponse[cmd], nil
}
