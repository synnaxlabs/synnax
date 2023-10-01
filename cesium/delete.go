// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package cesium

func DeleteChannel(dirname string, channel_to_remove string, opts ...Option) error {
	o := newOptions(dirname, opts...)
	if err := openFS(o); err != nil {
		return err
	}

	if err := o.fs.Remove(channel_to_remove); err != nil {
		return err
	}

	return nil
}
