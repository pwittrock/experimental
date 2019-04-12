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

package git

import (
	_ "k8s.io/api/core/v1"
	_ "k8s.io/apimachinery/pkg/runtime"
	_ "k8s.io/apimachinery/pkg/util/intstr"
)

type PullRequest struct {
	// Filter defines which PRs to match.
	Filter Filter `json:"filter,omitempty"`

	// SetPR defines what to publish to the PR.
	Set SetPR `json:"set,omitempty"`
}

type Issue struct {
	// Filter defines which PRs to match.
	Filter Filter `json:"filter,omitempty"`

	// SetPR defines what to publish to the PR.
	Set SetIssue `json:"set,omitempty"`
}

type Filter struct {
	// Milestone limits issues for the specified milestone. Possible values are
	// a milestone number, "none" for issues with no milestone, "*" for issues
	// with any milestone.
	Milestone string `json:"milestone,omitempty"`

	// State filters issues based on their state. Possible values are: open,
	// closed, all. Default is "open".
	State string `json:"state,omitempty"`

	// Assignee filters issues based on their assignee. Possible values are a
	// user name, "none" for issues that are not assigned, "*" for issues with
	// any assigned user.
	Assignee string `json:"assignee,omitempty"`

	// Creator filters issues based on their creator.
	Creator string `json:"creator,omitempty"`

	// Mentioned filters issues to those mentioned a specific user.
	Mentioned string `json:"mentioned,omitempty"`

	// Labels filters issues based on their label.
	Labels []string `json:"labels,omitempty,comma"`
}

type SetIssue struct {
	// Labels sets labels on the PR based on the roll out status if it is set.
	// +optional
	Labels *Labels `json:"labels,omitempty"`

	// StatusComment sets a comment on the PR based on the roll out status if it is set.
	// +optional
	StatusComment *Comment `json:"statusComment,omitempty"`
}

type SetPR struct {
	// Diff publishes a check to the pr with the diff results if it is set.
	// +optional
	Diff *Diff `json:"diff,omitempty"`

	// Lint publishes a check to the pr with the lint results if it is set.
	// +optional
	Lint *Lint `json:"lint,omitempty"`
}

type Remove struct {
	// Diff publishes a check to the pr with the diff results if it is set.
	// +optional
	Diff *Diff `json:"diff,omitempty"`

	// Lint publishes a check to the pr with the lint results if it is set.
	// +optional
	Lint *Lint `json:"lint,omitempty"`

	// Labels sets labels on the PR based on the roll out status if it is set.
	// +optional
	Labels *Labels `json:"labels,omitempty"`

	// StatusComment sets a comment on the PR based on the roll out status if it is set.
	// +optional
	StatusComment *Comment `json:"statusComment,omitempty"`
}

// StatusComment writes a comment to the PR with the status of the roll out.
// Continuously updates the same comment on the PR.
type Comment struct {
	// Header is text published at the top of the comment
	// +optional
	Header string `json:"header,omitempty"`
}

// Labels sets labels on the PR based on the status of the roll out.
type Labels struct {
	// +optional
	SetSuccess map[string]string `json:"setSucess,omitempty"`

	// +optional
	SetFailed map[string]string `json:"setFailed,omitempty"`

	// +optional
	SetInProgress map[string]string `json:"setInProgress,omitempty"`
}

// Lint publishes a check to the PR with lint results for the to-be-applied changes.
type Lint struct {
	// Command is the lint command to run.  Its output will be published.
	Command string `json:"command,omitempty"`

	// Args are the arguments to the lint command.
	// +optional
	Args string `json:"args,omitempty"`

	// Env contains environment variables for the Command
	// +optional
	Env map[string]string `json:"env,omitempty"`
}

// Diff publishes a check to the PR with the diff of the to-be-applied changes and
// what is live in the cluster.  If dry-run returns an error, will publish this instead.
type Diff struct {
	// IncludeSecretData will include the secret data in the diff if set to true, otherwise it will
	// remove secret data from the diff.  Never set this unless you have a really good reason!
	// +optional
	IncludeSecretData *bool `json:"includeSecretData,omitempty"`
}
