/*
Copyright 2018 The Tekton Authors
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
	"fmt"

	"github.com/spf13/cobra"
)

// gitCmd represents the git command
var gitCmd = &cobra.Command{
	Use:   "git",
	Short: "",
	Long:  `.`,
	Example: `
# Create your clusters and environments
$ tekctl git add test/us-west --cluster=us-west
$ tekctl git add staging/us-west --cluster=us-west
$ tekctl git add prod/us-west --cluster=us-west
$ tekctl git add prod/us-east --cluster=us-west

# Use your favorite tools to generate your yamls
...

# Promote master to Test environment
$ git tag -a v1.1 -m "v1.1 release"
$ git push remote v1.1
$ tekctl git promote tag:v1.1 test/
$ git push origin deploy-test # Applied through a PR review

# Promote Test to Staging
$ tekctl git promote test/ staging/
$ git push origin deploy-staging # Applied through a PR review

# Promote Staging to Prod
$ tekctl git promote test/ prod/
$ git push origin deploy-prod # Applied through a PR review

# Rollback Prod
$ tekctl git rollback prod
$ git push origin deploy-prod # Applied through a PR review

# Rollback Prod to a specific tag
$ tekctl git rollback prod --to-tag v0.9
$ git push origin deploy-prod # Applied through a PR review


# Modify the the prod us-west instance of the foo Deployment in the bar namespace.
# Generate an overlay for only that environment.
$ tekctl git customize prod/us-west deploy.apps foo -n bar
$ git commit -m "Increase memory of foo in us-west"
$ git push origin deploy-prod # Applied through a PR review
`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("git called")
	},
}
