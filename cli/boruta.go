package cli

import (
	"github.com/SamsungSLAV/slav/logger"
	"github.com/spf13/cobra"
)

type borutaCmd struct {
	Command *cobra.Command
}

func NewBorutaCmd() *borutaCmd {
	return &borutaCmd{
		Command: &cobra.Command{
			Use:   "boruta",
			Short: "Boruta command line interface",
			Long:  "", //TODO
			Run: func(cmd *cobra.Command, args []string) {
				err := cmd.Usage()
				if err != nil {
					logger.WithError(err).Error("Failed to print boruta command usage.")
				}
			},
		},
	}
}
