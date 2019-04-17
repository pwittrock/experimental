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
var p *int32
var path *string
var refs, orgs, repos *[]string

func GetCommand() *cobra.Command {
	c := &cobra.Command{
		Use:   "events",
		Short: "Listen for Events and trigger TaskRuns. ",
		Long:  ``,
		RunE:  RunE,
	}
	wiregithub.WebhookFlags(c)
	p = c.Flags().Int32("port", 8080, "port to listen for webhook events on.")
	refs = c.Flags().StringSlice("ref", []string{}, "")
	orgs = c.Flags().StringSlice("org", []string{}, "")
	repos = c.Flags().StringSlice("repo", []string{}, "")
	path = c.Flags().String("path", "tekton", "")
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
		fmt.Printf("error validating request: %v\n%+v\n", err, r)
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
		fmt.Printf("error doing %s: %v\n", op, err)
		return nil
	}

	objs, err := s.GetResources(event, filepath.Join(path, op, "*.yaml"), "push")
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
	ref := event.GetRef()
	if len(event.GetBaseRef()) > 0 {
		ref = event.GetBaseRef()
	}

	dir, err := ioutil.TempDir(os.TempDir(), "git-clone")
	if err != nil {
		return "", nil, err
	}
	clean := func() { os.RemoveAll(dir) } // clean up

	err = os.Chdir(dir)
	if err != nil {
		return "", clean, err
	}

	r := fmt.Sprintf("https://github.com/%s.git", event.GetRepo().GetFullName())
	loc := filepath.Join(dir, event.GetRepo().GetName())

	_, err = gitv4.PlainClone(loc, false, &gitv4.CloneOptions{
		URL:           r,
		Progress:      os.Stdout,
		Depth:         1,
		ReferenceName: plumbing.ReferenceName(ref),
		Auth: &httpv4.BasicAuth{
			Username: "", // anything except an empty string
			Password: string(s.Secret),
		},
	})
	if err != nil {
		return "", clean, err
	}

	fmt.Printf("cloned %s into %s\n", r, loc)

	fmt.Printf("reading files...\n")
	files, err := ioutil.ReadDir(loc)
	if err != nil {
		return "", clean, err
	}
	for i := range files {
		file := files[i]
		fmt.Printf("cloned file: %s\n", file.Name())
	}
	return filepath.Join(loc, *path), clean, nil
}

func (s *GitHubEventMonitor) GetResources(event *github.PushEvent, path, trigger string) ([]*unstructured.Unstructured, error) {
	ref := event.GetRef()
	if len(event.GetBaseRef()) > 0 {
		ref = event.GetBaseRef()
	}

	fmt.Printf("parsing template %s\n", path)
	t, err := template.ParseGlob(path)
	if err != nil {
		return nil, err
	}
	buf := &bytes.Buffer{}
	for _, tmpl := range t.Templates() {
		err = tmpl.Execute(buf, Data{
			Ref: strings.Replace(ref, "refs/", "", -1),
			URL: fmt.Sprintf("https://github.com/%s", event.Repo.GetFullName()),
		})
		if err != nil {
			return nil, err
		}
		buf.WriteString("\n---\n")
	}

	fmt.Printf("found runs\n%s\n", buf.String())
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
		config := configs[i]
		fmt.Printf("found object %s %s\n", config.GetName(), config.GetGenerateName())
		if v, found := config.GetAnnotations()["tekctl.tektoncd.dev/trigger"]; found {
			for _, p := range strings.Split(v, ",") {
				if p == trigger {
					match = append(match, config)
					break
				}
			}
		}
	}

	return match, nil
}

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
	Ref string
	URL string
}
