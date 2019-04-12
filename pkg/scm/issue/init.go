package issue

import (
	"github.com/google/wire"
	"github.com/spf13/cobra"
	"tektoncd.dev/experimental/pkg/deprecated/status"
)

var ProviderSet = wire.NewSet(Updater{}, NewLabelsFlag, status.Lister{}, status.Provider{})
var labelsFlag = Labels{}

func Flags(command *cobra.Command) {
	command.Flags().StringSliceVar(&labelsFlag.AddInProgress, "labels-add-in-progress", []string{}, "")
	command.Flags().StringSliceVar(&labelsFlag.AddComplete, "labels-add-complete", []string{}, "")
	command.Flags().StringSliceVar(&labelsFlag.AddFailed, "labels-add-failed", []string{}, "")
	command.Flags().StringSliceVar(&labelsFlag.DeleteInProgress, "labels-delete-in-progress", []string{}, "")
	command.Flags().StringSliceVar(&labelsFlag.DeleteComplete, "labels-delete-complete", []string{}, "")
	command.Flags().StringSliceVar(&labelsFlag.DeleteFailed, "labels-delete-failed", []string{}, "")
}

func NewLabelsFlag() Labels {
	return labelsFlag
}
