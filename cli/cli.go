/*
 *  Copyright (c) 2017-2018 Samsung Electronics Co., Ltd All Rights Reserved
 *
 *  Licensed under the Apache License, Version 2.0 (the "License");
 *  you may not use this file except in compliance with the License.
 *  You may obtain a copy of the License at
 *
 *      http://www.apache.org/licenses/LICENSE-2.0
 *
 *  Unless required by applicable law or agreed to in writing, software
 *  distributed under the License is distributed on an "AS IS" BASIS,
 *  WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 *  See the License for the specific language governing permissions and
 *  limitations under the License
 */

// Package cli contains definitions of all boruta commands
package cli

// cli.go - common part for cli package.

import (
	"time"

	"github.com/spf13/cobra"

	"github.com/SamsungSLAV/boruta/cli/config"
	"github.com/SamsungSLAV/boruta/http/client"
)

const (
	// TimeFormat used in leszy cli.
	TimeFormat = time.RFC3339
)

type CommandsCommon struct {
	Command *cobra.Command
	Client  *client.BorutaClient
	// server is used to create boruta client instance in a handler
	config.Config
}
