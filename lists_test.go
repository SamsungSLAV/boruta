/*
 *  Copyright (c) 2018 Samsung Electronics Co., Ltd All Rights Reserved
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

package boruta

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSortOrderMarshalText(t *testing.T) {
	assert := assert.New(t)

	order := new(SortOrder)
	text, err := order.MarshalText()
	assert.Nil(err)
	assert.Equal(SortOrderAsc.String(), string(text))

	*order = SortOrderDesc
	text, err = order.MarshalText()
	assert.Nil(err)
	assert.Equal(SortOrderDesc.String(), string(text))

	*order = SortOrderAsc
	text, err = order.MarshalText()
	assert.Nil(err)
	assert.Equal(SortOrderAsc.String(), string(text))
}

func TestSortOrderUnmarshalText(t *testing.T) {
	assert := assert.New(t)

	order := new(SortOrder)
	err := order.UnmarshalText([]byte(SortOrderDesc.String()))
	assert.Nil(err)
	assert.Equal(SortOrderDesc, *order)

	err = order.UnmarshalText([]byte(SortOrderAsc.String()))
	assert.Nil(err)
	assert.Equal(SortOrderAsc, *order)

	*order = SortOrderDesc
	err = order.UnmarshalText([]byte(""))
	assert.Nil(err)
	assert.Equal(SortOrderAsc, *order)

	err = order.UnmarshalText([]byte("foo"))
	assert.Equal(ErrWrongSortOrder, err)
}

func TestListDirectionMarshalText(t *testing.T) {
	assert := assert.New(t)

	direction := new(ListDirection)
	text, err := direction.MarshalText()
	assert.Nil(err)
	assert.Equal(DirectionForward.String(), string(text))

	*direction = DirectionBackward
	text, err = direction.MarshalText()
	assert.Nil(err)
	assert.Equal(DirectionBackward.String(), string(text))

	*direction = DirectionForward
	text, err = direction.MarshalText()
	assert.Nil(err)
	assert.Equal(DirectionForward.String(), string(text))
}

func TestListDirectionUnmarshalText(t *testing.T) {
	assert := assert.New(t)

	direction := new(ListDirection)
	err := direction.UnmarshalText([]byte(DirectionBackward.String()))
	assert.Nil(err)
	assert.Equal(DirectionBackward, *direction)

	err = direction.UnmarshalText([]byte(DirectionForward.String()))
	assert.Nil(err)
	assert.Equal(DirectionForward, *direction)

	*direction = DirectionBackward
	err = direction.UnmarshalText([]byte(""))
	assert.Nil(err)
	assert.Equal(DirectionForward, *direction)

	err = direction.UnmarshalText([]byte("foo"))
	assert.Equal(ErrWrongListDirection, err)
}
