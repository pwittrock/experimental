// Copyright © 2019 NAME HERE <EMAIL ADDRESS>
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package triggers

import (
	"fmt"
	"net/http"
	"os"

	"github.com/google/go-github/github"
	"github.com/google/wire"
	"github.com/spf13/cobra"
	"tektoncd.dev/experimental/pkg/cligithub"
	"tektoncd.dev/experimental/pkg/wirecli"
	"tektoncd.dev/experimental/pkg/wirecli/wiregithub"
	// "github.com/cloudevents/sdk-go/pkg/cloudevents"
	// "github.com/cloudevents/sdk-go/pkg/cloudevents/client"
)

var ProviderSet = wire.NewSet(wirecli.ProviderSet, GitHubEventMonitor{})

func GetCommand() *cobra.Command {
	c := &cobra.Command{
		Use:   "events",
		Short: "Listen for Events and trigger TaskRuns. ",
		Long:  ``,
		RunE:  RunE,
	}
	wiregithub.WebhookFlags(c)
	return c
}

func RunE(cmd *cobra.Command, args []string) error {
	t, err := InitializeTrigger()
	if err != nil {
		return err
	}
	http.HandleFunc("/", t.ServeHTTP)
	return http.ListenAndServe(":8080", nil)
}

type GitHubEventMonitor struct {
	Secret cligithub.GitHubWebHookSecret
}

func (s *GitHubEventMonitor) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	payload, err := github.ValidatePayload(r, []byte(s.Secret))
	if err != nil {
		fmt.Fprintf(os.Stderr, "error validating webhook payload: %v\n", err)
		return
	}
	event, err := github.ParseWebHook(github.WebHookType(r), payload)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error parsing webhook payload: %v\n", err)
		return
	}
	switch event := event.(type) {
	case *github.RepositoryEvent:
		fmt.Printf("%v\n", event)
	}
}
