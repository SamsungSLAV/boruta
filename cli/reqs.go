package cli

import (
	"github.com/spf13/cobra"

	"github.com/SamsungSLAV/slav/logger"
)

type ReqsCmd struct {
	Command *cobra.Command
}

func NewReqsCmd() *ReqsCmd {
	return &ReqsCmd{
		Command: &cobra.Command{
			Use:   "reqs",
			Short: "Boruta requests management.",
			Long:  "", //TODO
			Run: func(cmd *cobra.Command, args []string) {
				err := cmd.Usage()
				if err != nil {
					logger.WithError(err).Error("Failed to print reqs command usage.")
				}
			},
		},
	}

}
