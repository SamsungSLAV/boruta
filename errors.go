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

// File errors.go provides error types that may occur in more than one component.

package boruta

import (
	"errors"
	"fmt"
)

// NotFoundError is used whenever searched element is missing.
type NotFoundError string

func (err NotFoundError) Error() string {
	return fmt.Sprintf("%s not found", string(err))
}

var (
	// ErrInternalLogicError means that boruta's implementation has detected unexpected behaviour.
	ErrInternalLogicError = errors.New("Boruta's internal logic error")
	// ErrWrongSortItem means that unknown item name was provided.
	ErrWrongSortItem = errors.New("unknown name of item by which list should be sorted")
	// ErrWrongSortOrder is returned when user provided unknown order.
	ErrWrongSortOrder = errors.New("unknown sort order (valid: ascending or descending)")
)
