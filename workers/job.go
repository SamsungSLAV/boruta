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

package workers

import (
	"github.com/SamsungSLAV/boruta"
	"github.com/SamsungSLAV/boruta/tunnels"
)

// Job describes worker job.
type Job struct {
	// Access describes details of the connection to Dryad. It is returned to the request
	// owner when a job for request is run and acquired by the user.
	Access boruta.AccessInfo
	// Tunnel is a connection to Dryad for the user.
	Tunnel tunnels.Tunneler
	// Req is ID of the worked request.
	Req boruta.ReqID
}
