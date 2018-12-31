package cli

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/SamsungSLAV/boruta"
	"github.com/SamsungSLAV/slav/logger"
)

var (
	defaultValidAfterStr   = "now"
	defaultRequestDuration = "4h"
	defaultDeadlineStr     = defaultRequestDuration + " from now"
)

type ReqsNewCmd struct {
	Command    *cobra.Command
	Requests   boruta.Requests
	caps       map[string]string
	prio       uint8
	validAfter string
	deadline   string
	pretty     bool
}

func NewReqsNewCmd(r boruta.Requests) *ReqsNewCmd {
	rc := &ReqsNewCmd{}
	rc.Requests = r
	rc.Command = &cobra.Command{
		Use:   "new",
		Short: "New Boruta request.",
		Long:  "", //TODO
		Run:   rc.NewReqs,
	}
	rc.addFlags()
	return rc
}

func (rc *ReqsNewCmd) addFlags() {
	flagset := rc.Command.Flags()
	flagset.StringToStringVarP(&rc.caps, "caps", "c", nil,
		"capabilities of requested device, e.g.: device_type=rpi3,arch=armv7") //TODO: suppress showing nil default
	flagset.Uint8VarP(&rc.prio, "priority", "p", 8,
		"importance of the request. Lower is more important")
	flagset.StringVarP(&rc.validAfter, "valid-after", "v", defaultValidAfterStr,
		"point in time, after which request will be executed (if device is avaliable). Should be"+
			" specified in following format: "+TimeFormat)
	flagset.StringVarP(&rc.deadline, "deadline", "d", defaultDeadlineStr,
		"point in time, after which request will TIMEOUT if it has not been fulfilled.") //TODO
	flagset.BoolVar(&rc.pretty, "pretty", true, "control prettyfing jsons.")
}

func (rc *ReqsNewCmd) NewReqs(cmd *cobra.Command, args []string) {
	// set defaults
	validAfter := time.Now()
	reqDuration, err := time.ParseDuration(defaultRequestDuration)
	if err != nil {
		logger.WithError(err).Error("Failed to parse default request duration.")
	}
	deadline := time.Now().Add(reqDuration)

	// parse input
	if rc.validAfter != defaultValidAfterStr {
		if validAfter, err = time.Parse(TimeFormat, rc.validAfter); err != nil {
			logger.WithError(err).Error("Failed to parse time. Correct format is: " + TimeFormat)
		}
	}
	if rc.deadline != defaultDeadlineStr {
		if deadline, err = time.Parse(TimeFormat, rc.deadline); err != nil {
			logger.WithError(err).Error("Failed to parse time. Correct format is: " + TimeFormat)
		}
	}
	// request data
	//TODO: require at least device_type cap

	rid, err := rc.Requests.NewRequest(boruta.Capabilities(rc.caps), boruta.Priority(rc.prio),
		boruta.UserInfo{}, validAfter, deadline)
	if err != nil {
		logger.WithError(err).Error("Failed to create request.")
	}
	// no need to marshal one value to json.
	fmt.Println(rid)
}
