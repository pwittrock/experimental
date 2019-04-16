/*
Copyright 2019 The Tekton Authors
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

package cligithub

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/google/go-github/github"
	"gopkg.in/src-d/go-git.v4/plumbing/object"
)

type GitOwnerRepo string
type GitOwner string
type GitRepo string
type Name string

type IssueClient struct {
	Client *github.Client
	Repo   GitRepo
	Owner  GitOwner
	Name   Name
}

var mergeCommitRegex = regexp.MustCompile("^Merge pull request #(\\d+) from")

func (ic *IssueClient) GetPRIssue(commit *object.Commit) (*github.Issue, error) {
	if commit == nil {
		return nil, fmt.Errorf("missing commit")
	}
	if commit.Message == "" {
		return nil, fmt.Errorf("missing commit message")
	}
	matches := mergeCommitRegex.FindStringSubmatch(commit.Message)
	if len(matches) < 2 {
		return nil, fmt.Errorf("latest commit has no PR - message [%s]", strings.TrimSpace(commit.Message))
	}
	num, err := strconv.Atoi(matches[1])
	if err != nil {
		return nil, fmt.Errorf("latest commit has no PR: %v", err)
	}

	prIssue, _, err := ic.Client.Issues.Get(context.Background(), string(ic.Owner), string(ic.Repo), num)
	return prIssue, err
}

var prBodyRegex = regexp.MustCompile("rollout-status: #(\\d+)\\n\\s+")

func (ic *IssueClient) GetReleaseIssue(pr *github.Issue) (*github.Issue, error) {
	body := pr.GetBody()
	var err error
	var statusIssueNum int
	if !prBodyRegex.Match([]byte(body)) {
		// Create an Issue
		title := fmt.Sprintf("Deploy %d", pr.GetNumber())
		i, _, err := ic.Client.Issues.Create(
			context.Background(), string(ic.Owner), string(ic.Repo), &github.IssueRequest{Title: &title})
		if err != nil {
			return nil, err
		}

		// Update the PR
		body := fmt.Sprintf("rollout-status: #%d\n\n%s", i.GetNumber(), pr.GetBody())
		_, _, err = ic.Client.Issues.Edit(context.Background(), string(ic.Owner), string(ic.Repo), pr.GetNumber(),
			&github.IssueRequest{Body: &body})
		if err != nil {
			return nil, err
		}
		statusIssueNum = i.GetNumber()
	} else {
		matches := prBodyRegex.FindStringSubmatch(pr.GetBody())
		statusIssueNum, err = strconv.Atoi(matches[1])
		if err != nil {
			return nil, err
		}
	}

	issue, _, err := ic.Client.Issues.Get(context.Background(), string(ic.Owner), string(ic.Repo), statusIssueNum)
	return issue, err
}

func (ic *IssueClient) GetReleaseComment(pr *github.Issue) (*github.IssueComment, error) {
	var commentRegex = regexp.MustCompile(fmt.Sprintf("^# %s ", string(ic.Name)))
	comments, _, err := ic.Client.Issues.ListComments(
		context.Background(),
		string(ic.Owner), string(ic.Repo), pr.GetNumber(), &github.IssueListCommentsOptions{})
	if err != nil {
		return nil, err
	}
	for _, comment := range comments {
		if commentRegex.Match([]byte(comment.GetBody())) {
			return comment, nil
		}
	}

	b := fmt.Sprintf("Rollout: %s", string(ic.Name))
	i, _, err := ic.Client.Issues.CreateComment(
		context.Background(),
		string(ic.Owner), string(ic.Repo), pr.GetNumber(), &github.IssueComment{Body: &b})
	if err != nil {
		return nil, err
	}
	return i, err
}

type GitHubWebHookSecretPath string
type GitHubWebHookSecret string
type GitHubTokenPath string
type GitHubToken string
