package cli

import (
	"github.com/spf13/cobra"

	"github.com/SamsungSLAV/slav/logger"
)

type WorkersCmd struct {
	Command *cobra.Command
}

func NewWorkersCmd() *WorkersCmd {
	return &WorkersCmd{
		Command: &cobra.Command{
			Use:   "workers",
			Short: "Boruta workers management.",
			Long:  "", //TODO
			Run: func(cmd *cobra.Command, args []string) {
				err := cmd.Usage()
				if err != nil {
					logger.WithError(err).Error("Failed to print workers command usage.")
				}
			},
		},
	}

}
