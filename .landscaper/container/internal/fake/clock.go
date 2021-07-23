// Copyright (c) 2021 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file
// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package fake

import "time"

// FakeClock implements gardenlogin.Clock interface
type FakeClock struct {
	FakeTime time.Time
}

// Now returns a constant time
func (f *FakeClock) Now() time.Time {
	return f.FakeTime
}

// NewFakeClock returns a fakeClock that can be used in tests
func NewFakeClock() (*FakeClock, error) {
	now, err := FakeNow()
	if err != nil {
		return nil, err
	}
	return &FakeClock{FakeTime: now}, nil
}

// FakeNow returns a constant time
func FakeNow() (time.Time, error) {
	return time.Parse(time.RFC3339, "2017-12-14T23:34:00.000Z")
}
