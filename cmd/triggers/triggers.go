/*
Copyright 2019 The Tekton Authors.
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at
    http://www.apache.org/licenses/LICENSE-2.0
Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package triggers

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"

	// "os/exec"

	"github.com/google/go-github/github"
	"github.com/google/wire"
	"github.com/spf13/cobra"
	gitv4 "gopkg.in/src-d/go-git.v4"
	"gopkg.in/src-d/go-git.v4/plumbing"
	httpv4 "gopkg.in/src-d/go-git.v4/plumbing/transport/http"

	"tektoncd.dev/experimental/pkg/cligithub"
	"tektoncd.dev/experimental/pkg/wirecli"
	"tektoncd.dev/experimental/pkg/wirecli/wiregithub"
	// "github.com/cloudevents/sdk-go/pkg/cloudevents"
	// "github.com/cloudevents/sdk-go/pkg/cloudevents/client"
)

var ProviderSet = wire.NewSet(wirecli.ProviderSet, GitHubEventMonitor{})
var p *int32
var refs *[]string

func GetCommand() *cobra.Command {
	c := &cobra.Command{
		Use:   "events",
		Short: "Listen for Events and trigger TaskRuns. ",
		Long:  ``,
		RunE:  RunE,
	}
	wiregithub.WebhookFlags(c)
	p = c.Flags().Int32("port", 8080, "port to listen for webhook events on.")
	refs = c.Flags().StringSlice("refs", []string{}, ".")
	return c
}

func RunE(cmd *cobra.Command, args []string) error {
	t, err := InitializeTrigger()
	if err != nil {
		return err
	}
	fmt.Printf("listening on port %d...\n", *p)
	return http.ListenAndServe(fmt.Sprintf(":%d", *p), t)
}

type GitHubEventMonitor struct {
	Secret cligithub.GitHubWebHookSecret
	Token  cligithub.GitHubToken
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
	case *github.PushEvent:
		err = s.DoPush(event)
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
	}

}

func (s *GitHubEventMonitor) DoPush(event *github.PushEvent) error {
	if len(*refs) > 0 {
		found := false
		for _, ref := range *refs {
			if ref == *event.Ref {
				found = true
			}
		}
		if !found {
			return nil
		}
	}

	fmt.Printf("=====\n%+v\n=====\n", event)
	fmt.Printf("name: %s\n", event.Repo.GetFullName())

	dir, err := ioutil.TempDir(os.TempDir(), "git-clone")
	if err != nil {
		return err
	}
	defer os.RemoveAll(dir) // clean up

	err = os.Chdir(dir)
	if err != nil {
		return err
	}

	r := fmt.Sprintf("https://github.com/%s.git", event.GetRepo().GetFullName())
	loc := filepath.Join(dir, event.GetRepo().GetName())
	fmt.Printf("cloning repository %s into %s\n", r, loc)

	_, err = gitv4.PlainClone(loc, false, &gitv4.CloneOptions{
		URL:           r,
		Progress:      os.Stdout,
		Depth:         1,
		ReferenceName: plumbing.ReferenceName(event.GetRef()),
		Auth: &httpv4.BasicAuth{
			Username: "", // anything except an empty string
			Password: string(s.Secret),
		},
	})
	if err != nil {
		return err
	}

	fmt.Printf("cloned https://github.com/%s\n", event.GetRepo().GetFullName())

	fmt.Printf("reading files...\n")
	files, err := ioutil.ReadDir(loc)
	if err != nil {
		return err
	}
	for i := range files {
		file := files[i]
		fmt.Printf("cloned file: %s\n", file.Name())
	}

	tekPath := filepath.Join(loc, "tekton")

	if _, err := os.Stat(tekPath); os.IsNotExist(err) {
		fmt.Printf("missing tekton directory\n")
		return nil
	}

	cfgPath := filepath.Join(tekPath, "config")
	if _, err := os.Stat(cfgPath); err == nil {
		fmt.Printf("applying config...\n")
		cmd := exec.Command("kubectl", "apply", "--filename", cfgPath, "--recursive")
		cmd.Stderr = os.Stderr
		cmd.Stdout = os.Stdout
		err = cmd.Run()
		if err != nil {
			return err
		}
	}

	runsPath := filepath.Join(tekPath, "runs")
	if _, err := os.Stat(runsPath); err == nil {
		fmt.Printf("creating runs...\n")
		files, err := ioutil.ReadDir(runsPath)
		if err != nil {
			return err
		}
		for i := range files {
			file := files[i]
			fmt.Printf("cloned file: %s\n", file.Name())
		}
	}

	fmt.Printf("done\n")
	return nil

}
