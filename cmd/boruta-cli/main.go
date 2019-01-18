package main

import (
	//"github.com/spf13/viper"

	"github.com/SamsungSLAV/boruta/cli"
	"github.com/SamsungSLAV/boruta/http/client"
)

func main() {
	// check boruta cli config
	// if empty- generate it and ask interactively for boruta addr
	// integrate viper and bind viper output to cobra flags propagated through whole app

	client := client.NewBorutaClient("http://106.120.47.240:8487")

	//define commands
	borutaCmd := cli.NewBorutaCmd()
	reqsCmd := cli.NewReqsCmd()
	workersCmd := cli.NewWorkersCmd()

	// register commands
	borutaCmd.Command.AddCommand(
		reqsCmd.Command,
		workersCmd.Command,
	)
	reqsCmd.Command.AddCommand(
		cli.NewReqsListCmd(client).Command,
		cli.NewReqsNewCmd(client).Command,
		cli.NewReqsCloseCmd(client).Command,
		cli.NewReqsAcquireCmd(client).Command,
	)
	/*	workersCmd.Command.AddCommand(
			cli.NewWorkersListCmd(client).Command,
			cli.NewWorkersSetStateCmd(client).Command,
			cli.NewWorkersSetGroupsCmd(client).Command,
			cli.NewWorkersDeregisterCmd(client).Command,
		)
	*/
	borutaCmd.Command.Execute()
}
