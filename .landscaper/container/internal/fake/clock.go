// Copyright (c) 2021 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file
// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package fake

import "time"

// Clock implements gardenlogin.Clock interface
type Clock struct {
	FakeTime time.Time
}

// Now returns a constant time
func (f *Clock) Now() time.Time {
	return f.FakeTime
}

// NewClock returns a fakeClock that can be used in tests
func NewClock() (*Clock, error) {
	now, err := Now()
	if err != nil {
		return nil, err
	}

	return &Clock{FakeTime: now}, nil
}

// Now returns a constant time
func Now() (time.Time, error) {
	return time.Parse(time.RFC3339, "2017-12-14T23:34:00.000Z")
}
