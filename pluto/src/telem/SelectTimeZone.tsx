// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Select } from "@synnaxlabs/lyra/select";
import { type TimeZone } from "@synnaxlabs/x";
import { type ReactElement } from "react";

export interface SelectTimeZoneProps extends Select.ButtonsProps<TimeZone> {}

export const SelectTimeZone = (props: SelectTimeZoneProps): ReactElement => (
  <Select.Buttons {...props}>
    <Select.Item itemKey="UTC" tooltip="UTC">
      UTC
    </Select.Item>
    <Select.Item itemKey="local" tooltip="Local time zone">
      Local
    </Select.Item>
  </Select.Buttons>
);
