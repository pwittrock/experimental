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
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/google/go-github/github"
	"github.com/google/wire"
	"github.com/spf13/cobra"
	gitv4 "gopkg.in/src-d/go-git.v4"
	"gopkg.in/src-d/go-git.v4/plumbing"
	httpv4 "gopkg.in/src-d/go-git.v4/plumbing/transport/http"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/yaml"
	"tektoncd.dev/experimental/pkg/cligithub"
	"tektoncd.dev/experimental/pkg/wirecli"
	"tektoncd.dev/experimental/pkg/wirecli/wiregithub"
)

var ProviderSet = wire.NewSet(wirecli.ProviderSet, GitHubEventMonitor{})

var port *int32
var path, tektonBranch, tektonRepo *string
var triggerRefWhitelist *[]string

func GetCommand() *cobra.Command {
	c := &cobra.Command{
		Use:   "events",
		Short: "Listen for Events and trigger TaskRuns. ",
		Long:  ``,
		RunE:  RunE,
	}
	wiregithub.WebhookFlags(c)
	port = c.Flags().Int32("port", 8080, "port to listen for webhook events on.")
	triggerRefWhitelist = c.Flags().StringSlice("refs", []string{"refs/heads/", "refs/tags/"}, "if not empty, white list triggers to these ref prefixes")
	tektonBranch = c.Flags().String("tekton-branch", "tekton", "if not empty, use this branch for the Tekton config.")
	tektonRepo = c.Flags().String("tekton-repo", "", "if not empty, use this repo for the Tekton config.  e.g. tektoncd/experimental")
	path = c.Flags().String("path", "tekton", "look for Tekton configs in this directory")
	return c
}

func RunE(cmd *cobra.Command, args []string) error {
	t, err := InitializeTrigger()
	if err != nil {
		return err
	}
	fmt.Printf("listening on port %d...\n", *port)
	return http.ListenAndServe(fmt.Sprintf(":%d", *port), t)
}

type GitHubEventMonitor struct {
	Secret cligithub.GitHubWebHookSecret
	Token  cligithub.GitHubToken
}

func (s *GitHubEventMonitor) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	payload, err := github.ValidatePayload(r, []byte(s.Secret))
	if err != nil {
		return
	}
	event, err := github.ParseWebHook(github.WebHookType(r), payload)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error parsing webhook payload: %v\n", err)
		return
	}

	switch event := event.(type) {
	case *github.PushEvent:
		err = s.DoPushEvent(event)
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
	}

	fmt.Printf("=====Message=====\n%+v\n=====\n\n", event)

}

func (s *GitHubEventMonitor) DoPushEvent(event *github.PushEvent) error {
	path, clean, err := s.DoPushClone(event)
	if clean != nil {
		defer clean()
	}
	if err != nil {
		return err
	}

	if err := s.DoPushDir(event, path, "apply"); err != nil {
		return err
	}
	if err := s.DoPushDir(event, path, "create"); err != nil {
		return err
	}

	fmt.Printf("done\n")
	return nil
}

func (s *GitHubEventMonitor) DoPushDir(event *github.PushEvent, path, op string) error {
	if _, err := os.Stat(filepath.Join(path, op)); err != nil {
		fmt.Printf("skipping %s\n", op)
		return nil
	}

	objs, err := s.GetResources(event, filepath.Join(path, op, "*.yaml"))
	if err != nil {
		return err
	}
	err = s.DoKubectlAll(op, objs)
	if err != nil {
		return err
	}
	return nil
}

func (s *GitHubEventMonitor) DoPushClone(event *github.PushEvent) (string, func(), error) {
	dir, err := ioutil.TempDir(os.TempDir(), "git-clone")
	if err != nil {
		return "", nil, err
	}
	clean := func() { os.RemoveAll(dir) } // clean up

	err = os.Chdir(dir)
	if err != nil {
		return "", clean, err
	}

	repoName := strings.TrimSpace(*tektonRepo)
	if len(repoName) == 0 {
		repoName = event.GetRepo().GetFullName()
	}

	destDir := filepath.Join(dir, event.GetRepo().GetName())

	url := fmt.Sprintf("https://github.com/%s.git", repoName)
	fmt.Printf("cloning %s into %s\n", url, destDir)
	_, err = gitv4.PlainClone(destDir, false, &gitv4.CloneOptions{
		URL:           url,
		Progress:      os.Stdout,
		Depth:         1,
		ReferenceName: plumbing.NewBranchReferenceName(*tektonBranch),
		Auth: &httpv4.BasicAuth{
			Username: "", // anything except an empty string
			Password: string(s.Secret),
		},
	})
	if err != nil {
		return "", clean, err
	}

	return filepath.Join(destDir, *path), clean, nil
}

func (s *GitHubEventMonitor) GetResources(event *github.PushEvent, path string) ([]*unstructured.Unstructured, error) {
	// Check if ref is whitelisted
	found := false
	for _, match := range *triggerRefWhitelist {
		if strings.HasPrefix(event.GetRef(), match) {
			found = true
			break
		}
	}
	if !found {
		return nil, nil
	}

	t := template.New("configs").Funcs(template.FuncMap{
		"TrimPrefix": strings.TrimPrefix,
		"TrimSuffix": strings.TrimSuffix,
		"TrimSpace":  strings.TrimSpace,
	})
	t, err := t.ParseGlob(path)
	if err != nil {
		return nil, err
	}
	buf := &bytes.Buffer{}

	for _, tmpl := range t.Templates() {
		err = tmpl.Execute(buf, Data{
			Ref: event.GetRef(),
			URL: fmt.Sprintf("https://github.com/%s", event.Repo.GetFullName()),
		})
		if err != nil {
			return nil, err
		}
		buf.WriteString("\n---\n")
	}

	objs := strings.Split(string(buf.String()), "---")
	var configs, match []*unstructured.Unstructured
	for i := range objs {
		o := objs[i]
		if len(strings.TrimSpace(o)) == 0 {
			continue
		}
		body := map[string]interface{}{}
		if err := yaml.Unmarshal([]byte(o), &body); err != nil {
			return nil, err
		}
		configs = append(configs, &unstructured.Unstructured{Object: body})
	}

	for i := range configs {
		fmt.Printf("checking %s\n", configs[i].GetGenerateName())
		if !s.Check(configs[i], triggerAnnotation, "push", false) {
			fmt.Printf("doesn't match trigger\n")
			continue
		}
		if !s.CheckPrefix(configs[i], matchAnnotation, event.GetRef(), true) {
			fmt.Printf("doesn't match push-type\n")
			continue
		}
		fmt.Printf("create %s\n", configs[i].GetGenerateName())
		match = append(match, configs[i])
	}

	return match, nil
}

func (s *GitHubEventMonitor) Check(obj *unstructured.Unstructured, annotation, value string, d bool) bool {
	fi := d
	if v, found := obj.GetAnnotations()[annotation]; found {
		fi = false
		for _, p := range strings.Split(v, ",") {
			if p == value {
				fi = true
			}
		}
	}
	return fi
}

func (s *GitHubEventMonitor) CheckPrefix(obj *unstructured.Unstructured, annotation, value string, d bool) bool {
	fi := d
	if v, found := obj.GetAnnotations()[annotation]; found {
		fi = false
		for _, p := range strings.Split(v, ",") {
			if strings.HasPrefix(value, p) {
				fi = true
			}
		}
	}
	return fi
}

const triggerAnnotation = "tekctl.tektoncd.dev/triggers"
const matchAnnotation = "tekctl.tektoncd.dev/match"

func (s *GitHubEventMonitor) DoKubectlAll(c string, objs []*unstructured.Unstructured) error {
	for i := range objs {
		err := s.DoKubectl(c, objs[i])
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *GitHubEventMonitor) GetBuf(obj *unstructured.Unstructured) (*bytes.Buffer, error) {
	m, err := yaml.Marshal(obj)
	if err != nil {
		return nil, err
	}
	buf := &bytes.Buffer{}
	_, err = buf.Write(m)
	if err != nil {
		return nil, err
	}
	return buf, err
}

func (s *GitHubEventMonitor) DoKubectl(c string, obj *unstructured.Unstructured) error {
	b, err := s.GetBuf(obj)
	if err != nil {
		return err
	}

	cmd := exec.Command("kubectl", c, "--filename", "-")
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout
	cmd.Stdin = b
	return cmd.Run()
}

type Data struct {
	Ref    string
	URL    string
	Tag    string
	Branch string
}
