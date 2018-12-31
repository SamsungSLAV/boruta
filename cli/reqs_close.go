package cli

import (
	"strconv"

	"github.com/spf13/cobra"

	"github.com/SamsungSLAV/boruta"
	"github.com/SamsungSLAV/slav/logger"
)

type ReqsCloseCmd struct {
	Command  *cobra.Command
	Requests boruta.Requests
}

func NewReqsCloseCmd(r boruta.Requests) *ReqsCloseCmd {
	rc := &ReqsCloseCmd{}
	rc.Requests = r
	rc.Command = &cobra.Command{
		Use:   "close",
		Short: "Close Boruta request.",
		Long:  "", //TODO
		Args:  cobra.ExactArgs(1),
		Run:   rc.CloseReqs,
	}
	return rc
}

func (rc *ReqsCloseCmd) CloseReqs(cmd *cobra.Command, args []string) {
	rid, err := strconv.ParseUint(args[0], 10, 64)
	if err != nil {
		logger.WithError(err).Errorf("Failed to parse %s to uint64", args[0])
	}
	err = rc.Requests.CloseRequest(boruta.ReqID(rid))
	if err != nil {
		logger.WithError(err).Error("Failed to create request.")
	}
}
