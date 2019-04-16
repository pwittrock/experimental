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

package wiregithub

import (
	"context"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"strings"

	"github.com/google/go-github/github"
	"github.com/google/wire"
	"github.com/spf13/cobra"
	"golang.org/x/oauth2"
	"tektoncd.dev/experimental/pkg/cligithub"
	"tektoncd.dev/experimental/pkg/util"
)

var ProviderSet = wire.NewSet(NewGitHubClient, NewGitOwner, NewGitRepo, NewGitRepoFlag,
	NewGitHubUser, NewNameFlag, cligithub.IssueClient{}, NewGitHubWebHookSecret, NewGitHubWebHookSecretPath,
	NewGitHubToken, NewGitHubTokenPath)

func Flags(command *cobra.Command) {
	var path, home string
	home = util.HomeDir()
	if len(home) > 0 {
		path = filepath.Join(home, ".github", "token")
	} else {
		command.MarkFlagRequired("github-token")
	}

	command.Flags().StringVar(&gitHubTokenPathFlag, "github-token", path, "path to GitHub token file.")

	command.Flags().StringVar(&gitRepoFlag, "repo", "", "repository - e.g. kubernetes/kubectl")
	command.MarkFlagRequired("repo")

	command.Flags().StringVar(&gitNameFlag, "name", "", "")
	command.MarkFlagRequired("name")
}

func WebhookFlags(command *cobra.Command) {
	command.Flags().StringVar(&gitHubTokenPathFlag, "github-token", "/etc/tekctl/github/token", "path to GitHub token file.")
	command.Flags().StringVar(&gitWebHookSecretPathFlag, "webhook-secret", "/etc/tekctl/github/secret", "path to GitHub WebHook secret file.")
}

var gitHubTokenPathFlag string

func NewGitHubTokenPath() cligithub.GitHubTokenPath {
	return cligithub.GitHubTokenPath(gitHubTokenPathFlag)
}

func NewGitHubToken(path cligithub.GitHubTokenPath) (cligithub.GitHubToken, error) {
	d, err := ioutil.ReadFile(string(path))
	return cligithub.GitHubToken(d), err
}

func NewGitHubClient(dat cligithub.GitHubToken) *github.Client {
	ctx := context.Background()
	ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: string(dat)})
	tc := oauth2.NewClient(ctx, ts)
	gc := github.NewClient(tc)
	return gc
}

var gitRepoFlag string

func NewGitRepoFlag() cligithub.GitOwnerRepo {
	return cligithub.GitOwnerRepo(gitRepoFlag)
}

func NewGitRepo(or cligithub.GitOwnerRepo) (cligithub.GitRepo, error) {
	parts := strings.Split(string(or), "/")
	if len(parts) != 2 {
		return "", fmt.Errorf("%s must have exactly 2 parts separated by '/'", or)
	}
	return cligithub.GitRepo(parts[1]), nil
}

var gitNameFlag string

func NewNameFlag() cligithub.Name {
	return cligithub.Name(gitNameFlag)

}

func NewGitOwner(or cligithub.GitOwnerRepo) (cligithub.GitOwner, error) {
	parts := strings.Split(string(or), "/")
	if len(parts) != 2 {
		return "", fmt.Errorf("%s must have exactly 2 parts separated by '/'", or)
	}
	return cligithub.GitOwner(parts[0]), nil
}

func NewGitHubUser(ghc *github.Client) (*github.User, error) {
	user, _, err := ghc.Users.Get(context.Background(), "")
	return user, err
}

var gitWebHookSecretPathFlag string

func NewGitHubWebHookSecret(p cligithub.GitHubWebHookSecretPath) (cligithub.GitHubWebHookSecret, error) {
	sec, err := ioutil.ReadFile(string(p))
	if err != nil {
		return "", err
	}

	sec = []byte(strings.TrimSpace(string(sec)))
	return cligithub.GitHubWebHookSecret(sec), nil
}

func NewGitHubWebHookSecretPath() cligithub.GitHubWebHookSecretPath {
	return cligithub.GitHubWebHookSecretPath(gitWebHookSecretPathFlag)
}
