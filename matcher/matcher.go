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

// Package matcher provides interface and implementation for taking actions related
// to assigning requests to workers and reacting to requests time events.
package matcher

import (
	"git.tizen.org/tools/boruta"
)

// Matcher defines interface for objects that can be notified about events.
type Matcher interface {
	// Notify triggers action in the matcher. The ReqID slice contain set
	// of requests' IDs related to the event. The slice can be empty if the event
	// requires generic actions on all requests.
	Notify([]boruta.ReqID)
}
