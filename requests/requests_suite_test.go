/*
 *  Copyright (c) 2017-2022 Samsung Electronics Co., Ltd All Rights Reserved
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

package requests

import (
	"errors"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/suite"
)

var (
	zeroTime  time.Time
	now       = time.Now().UTC()
	yesterday = now.AddDate(0, 0, -1).UTC()
	tomorrow  = now.AddDate(0, 0, 1).UTC()
	nextWeek  = now.AddDate(0, 0, 7).UTC()
)

type RequestsTestSuite struct {
	suite.Suite
	ctrl    *gomock.Controller
	wm      *MockWorkersManager
	jm      *MockJobsManager
	testErr error
	rqueue  *ReqsCollection
}

func (s *RequestsTestSuite) SetupTest() {
	s.ctrl = gomock.NewController(s.T())
	s.wm = NewMockWorkersManager(s.ctrl)
	s.jm = NewMockJobsManager(s.ctrl)
	s.testErr = errors.New("Test Error")
	s.wm.EXPECT().SetChangeListener(gomock.Any())
	s.rqueue = NewRequestQueue(s.wm, s.jm)
}

func (s *RequestsTestSuite) TearDownTest() {
	s.rqueue.Finish()
	s.ctrl.Finish()
}

func TestRequestsTestSuite(t *testing.T) {
	suite.Run(t, new(RequestsTestSuite))
}
