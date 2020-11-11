// Copyright (c) 2020 InfraCloud Technologies
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

package utils

import (
	"fmt"
	"testing"
)

func TestGetClusterNameFromKubectlCmd(t *testing.T) {

	type test struct {
		input    string
		expected string
	}

	tests := []test{
		{input: "get pods --cluster-name=minikube", expected: "minikube"},
		{input: "--cluster-name minikube1", expected: "minikube1"},
		{input: "--cluster-name minikube2 -n default", expected: "minikube2"},
		{input: "--cluster-name minikube -n=default", expected: "minikube"},
		{input: "--cluster-name", expected: ""},
		{input: "--cluster-name ", expected: ""},
		{input: "--cluster-name=", expected: ""},
		{input: "", expected: ""},
		{input: "--cluster-nameminikube1", expected: ""},
	}

	for _, ts := range tests {
		got := GetClusterNameFromKubectlCmd(ts.input)
		if got != ts.expected {
			t.Errorf("expected: %v, got: %v", ts.expected, got)
		}
	}
}

func TestGetStringInYamlFormat(t *testing.T) {
	var header = "allowed verbs"
	var commands = map[string]bool{
		"api-versions": true,
	}
	expected := fmt.Sprintf(header + "\n  - api-versions\n")
	got := GetStringInYamlFormat(header, commands)
	if got != expected {
		t.Errorf("expected: %v, got: %v", expected, got)
	}
}

func TestContainsMethod(t *testing.T) {
	type input struct {
		array []string
		value string
	}
	type test struct {
		input    input
		expected bool
	}

	input1 := input{
		array: []string{"get", "logs"},
		value: "logs",
	}
	input2 := input{
		array: []string{"get", "logs"},
		value: "describe",
	}
	input3 := input{
		array: []string{"get", "Logs"},
		value: "logs",
	}
	tests := []test{
		{
			input:    input1,
			expected: true,
		},
		{
			input:    input2,
			expected: false,
		},
		{
			input:    input3,
			expected: true,
		},
	}

	for _, ts := range tests {
		got := Contains(ts.input.array, ts.input.value)
		if got != ts.expected {
			t.Errorf("expected: %v, got: %v", ts.expected, got)
		}
	}
}
