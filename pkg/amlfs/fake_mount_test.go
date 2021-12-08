/*
Copyright 2020 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package amlfs

import (
	"fmt"
	"reflect"
	"testing"

	mount "k8s.io/mount-utils"
)

func TestMountSensitive(t *testing.T) {
	tests := []struct {
		desc        string
		source      string
		target      string
		expectedErr error
	}{
		{
			desc:        "[Error] Mocked source error",
			source:      "ut-container",
			target:      targetTest,
			expectedErr: fmt.Errorf("fake MountSensitive: source error"),
		},
		{
			desc:        "[Error] Mocked target error",
			source:      "container",
			target:      "error_mount_sens",
			expectedErr: fmt.Errorf("fake MountSensitive: target error"),
		},
		{
			desc:        "[Error] Mocked target error",
			source:      "container",
			target:      "error_mount",
			expectedErr: nil,
		},
	}

	d := NewFakeDriver()
	fakeMounter := &fakeMounter{}
	d.mounter = &mount.SafeFormatAndMount{
		Interface: fakeMounter,
	}
	for _, test := range tests {
		err := d.mounter.MountSensitive(test.source, test.target, "", nil, nil)
		if !reflect.DeepEqual(err, test.expectedErr) {
			t.Errorf("actualErr: (%v), expectedErr: (%v)", err, test.expectedErr)
		}
	}
}

func TestIsLikelyNotMountPoint(t *testing.T) {
	tests := []struct {
		desc        string
		file        string
		expectedErr error
	}{
		{
			desc:        "[Error] Mocked file error",
			file:        "./error_is_likely_target",
			expectedErr: fmt.Errorf("fake IsLikelyNotMountPoint: fake error"),
		},
		{desc: "[Success] Successful run",
			file:        targetTest,
			expectedErr: nil,
		},
		{
			desc:        "[Success] Successful run not a mount",
			file:        "./false_is_likely_target",
			expectedErr: nil,
		},
	}

	d := NewFakeDriver()
	fakeMounter := &fakeMounter{}
	d.mounter = &mount.SafeFormatAndMount{
		Interface: fakeMounter,
	}
	for _, test := range tests {
		_, err := d.mounter.IsLikelyNotMountPoint(test.file)
		if !reflect.DeepEqual(err, test.expectedErr) {
			t.Errorf("Unexpected error: %v", err)
		}
	}
}
