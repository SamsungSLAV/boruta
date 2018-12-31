package cli

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/SamsungSLAV/boruta"
	"github.com/SamsungSLAV/slav/logger"
)

//TODO: adjust marshalling of time.Time to more human readable form?

type ReqsListCmd struct {
	Command  *cobra.Command
	Requests boruta.Requests
	pretty   bool
}

func NewReqsListCmd(r boruta.Requests) *ReqsListCmd {
	rc := &ReqsListCmd{}
	rc.Requests = r
	rc.Command = &cobra.Command{
		Use:   "list",
		Short: "List Boruta requests.",
		Long:  "", //TODO
		Run:   rc.ListReqs,
	}
	rc.addFlags()
	return rc
}

func (rc *ReqsListCmd) addFlags() {
	flagset := rc.Command.Flags()
	flagset.BoolVar(&rc.pretty, "pretty", true, "desc") //TODO: pretty should be global flag
}

func (rc *ReqsListCmd) ListReqs(cmd *cobra.Command, args []string) {
	rinfo, err := rc.Requests.ListRequests(nil)
	if err != nil {
		logger.WithError(err).Error("Failed to list requests.")
	}
	var data []byte
	if data, err = json.Marshal(rinfo); err != nil {
		logger.WithError(err).Error("Failed to marshal Boruta's response.")
	}
	if rc.pretty {
		if data, err = json.MarshalIndent(rinfo, "", "    "); err != nil {
			logger.WithError(err).Error("Failed to marshal Boruta's reponse")
		}
	}
	fmt.Println(string(data))
}
