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

type GitHubTokenPath string

var ProviderSet = wire.NewSet(NewGitHubClient, NewGitHubTokenPathFlag, NewGitOwner, NewGitRepo, NewGitRepoFlag,
	NewGitHubUser, NewNameFlag, NewGitHubWebHookSecret, cligithub.IssueClient{})
var gitHubTokenPathFlag string

func Flags(command *cobra.Command) {
	var secretPath, tokenPath, home string
	home = util.HomeDir()
	if len(home) > 0 {
		tokenPath = filepath.Join(home, ".github", "token")
		secretPath = filepath.Join(home, ".github", "secret")
	} else {
		command.MarkFlagRequired("githubtoken")
		command.MarkFlagRequired("secret")
	}

	command.Flags().StringVar(&gitHubTokenPathFlag, "githubtoken", tokenPath, "path to GitHub token file")

	command.Flags().StringVar(&gitWebHookSecretFlag, "secret", secretPath, "path to WebHook secret file")
	command.MarkFlagRequired("secret")
}

func WebhookFlags(command *cobra.Command) {
	var path, home string
	home = util.HomeDir()
	if len(home) > 0 {
		path = filepath.Join(home, ".github", "token")
	} else {
		path = ""
		command.MarkFlagRequired("githubtoken")
	}

	command.Flags().StringVar(&gitHubTokenPathFlag, "githubtoken", path, "path to GitHub token file")

	command.Flags().StringVar(&gitRepoFlag, "repo", "", "repository - e.g. kubernetes/kubectl")
	command.MarkFlagRequired("repo")

	command.Flags().StringVar(&gitNameFlag, "name", "", "")
	command.MarkFlagRequired("name")
}

func NewGitHubTokenPathFlag() GitHubTokenPath {
	return GitHubTokenPath(gitHubTokenPathFlag)
}

func NewGitHubClient(path GitHubTokenPath) (*github.Client, error) {
	dat, err := ioutil.ReadFile(string(path))
	if err != nil {
		return nil, fmt.Errorf("unable to create GitHub client: %v\n", err)
	}
	ctx := context.Background()
	ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: string(dat)})
	tc := oauth2.NewClient(ctx, ts)
	gc := github.NewClient(tc)
	if err != nil {
		return nil, fmt.Errorf("unable to add GitHub client: %v\n", err)
	}
	return gc, nil
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

var gitWebHookSecretFlag string

func NewGitHubWebHookSecret() cligithub.GitHubWebHookSecret {
	return cligithub.GitHubWebHookSecret([]byte(gitWebHookSecretFlag))
}
