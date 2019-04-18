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

package updateissue

import (
	"fmt"
	"os"

	"github.com/google/wire"
	"github.com/spf13/cobra"

	"tektoncd.dev/experimental/pkg/clik8s"
	"tektoncd.dev/experimental/pkg/scm/issue"
	"tektoncd.dev/experimental/pkg/wirecli"
	"tektoncd.dev/experimental/pkg/wirecli/wiregithub"
)

// ProviderSet captures the dependencies for this command.
var ProviderSet = wire.NewSet(wirecli.ProviderSet)

func GetCommand() *cobra.Command {
	// Create Command
	updateIssueCmd := &cobra.Command{
		Args:  cobra.ExactArgs(1),
		Use:   "update-issue",
		Short: "Create or update an issue for the most recently merged PR.",
		Long: `
Create or update an Issue for the most recently merged PR in the current git environment.

- Write the current rollout status to the Issue description.
- Add / remove labels to the Issue based on the rollout status.

# Create an Issue for the latest PR in the current git repo (from the CWD) and write the
# rollout status to its description.
tekctl scm update-issue --repo pwittrock/najena path/to/kustomization
`,
		RunE: func(cmd *cobra.Command, args []string) error {
			run, err := InitializeUpdater(clik8s.ResourceConfigPath(args[0]))

			if err != nil {
				return err
			}
			if err := run.Do(); err != nil {
				fmt.Fprintf(os.Stderr, "unable to perform issue update: %v\n", err)
				os.Exit(1)
			}
			return nil
		},
	}

	issue.Flags(updateIssueCmd)
	wiregithub.Flags(updateIssueCmd)
	return updateIssueCmd
}
