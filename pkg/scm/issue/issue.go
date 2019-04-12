package issue

import (
	"context"
	"fmt"
	"regexp"

	"github.com/google/go-github/github"
	"gopkg.in/src-d/go-git.v4/plumbing/object"
	"tektoncd.dev/experimental/pkg/cligithub"
	"tektoncd.dev/experimental/pkg/clik8s"
	"tektoncd.dev/experimental/pkg/deprecated/status"
	"tektoncd.dev/experimental/pkg/scm/issue/markdown"
)

// Match commit message for PR Merge Requests in GitHub
type MergeCommitRegex *regexp.Regexp
type PRBodyRegex *regexp.Regexp

type Updater struct {
	// GitHub
	Owner cligithub.GitOwner
	Repo  cligithub.GitRepo
	Name  cligithub.Name

	// Actions
	Labels Labels

	//
	Commit *object.Commit

	// Dependencies
	GHClient    *github.Client
	Lister      *status.Lister
	Resources   clik8s.ResourceConfigs
	IssueClient *cligithub.IssueClient
	Markdowner  markdown.Markdowner
}

type Labels struct {
	AddInProgress    []string
	DeleteInProgress []string

	AddComplete    []string
	DeleteComplete []string

	AddFailed    []string
	DeleteFailed []string
}

func (u *Updater) Do() error {
	prIssue, err := u.IssueClient.GetPRIssue(u.Commit)
	if err != nil {
		return err
	}

	releaseIssue, err := u.IssueClient.GetReleaseIssue(prIssue)
	if err != nil {
		return err
	}

	fmt.Printf("Updating Issue %v for PR %v\n", releaseIssue.GetNumber(), prIssue.GetNumber())
	return u.updateStatusIssue(releaseIssue)
}

func (u *Updater) updateStatusIssue(i *github.Issue) error {
	objs, err := status.UnstructuredToObjects(u.Resources)
	if err != nil {
		return err
	}

	if err := u.doInProgressLabels(i); err != nil {
		return err
	}
	for done := false; done != true; {
		if done, err = u.Lister.List(objs); err != nil {
			return err
		}

		md, err := u.Markdowner.GetMarkdown(objs)
		if err != nil {
			return err
		}

		c, err := u.IssueClient.GetReleaseComment(i)
		if err != nil {
			return err
		}
		_, _, err = u.GHClient.Issues.EditComment(
			context.Background(), string(u.Owner), string(u.Repo), c.GetID(), &github.IssueComment{Body: &md})
		if err != nil {
			return err
		}
	}
	if err := u.doCompleteLabels(i); err != nil {
		return err
	}

	return nil
}

func (u *Updater) doInProgressLabels(i *github.Issue) error {
	_, _, err := u.GHClient.Issues.AddLabelsToIssue(context.Background(),
		string(u.Owner), string(u.Repo), i.GetNumber(), u.Labels.AddInProgress)
	if err != nil {
		return err
	}
	for _, l := range u.Labels.DeleteInProgress {
		_, err = u.GHClient.Issues.RemoveLabelForIssue(context.Background(),
			string(u.Owner), string(u.Repo), i.GetNumber(), l)
		if err != nil {
			return err
		}
	}
	return nil
}

func (u *Updater) doCompleteLabels(i *github.Issue) error {
	_, _, err := u.GHClient.Issues.AddLabelsToIssue(context.Background(),
		string(u.Owner), string(u.Repo), i.GetNumber(), u.Labels.AddComplete)
	if err != nil {
		return err
	}
	for _, l := range u.Labels.DeleteComplete {
		_, err = u.GHClient.Issues.RemoveLabelForIssue(context.Background(),
			string(u.Owner), string(u.Repo), i.GetNumber(), l)
		if err != nil {
			return err
		}
	}
	return nil
}
