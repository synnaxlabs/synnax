// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { TIME_ZONES, type TimeZone } from "@synnaxlabs/x";
import { type ReactElement } from "react";

import { Select } from "@/select";

export interface SelectTimeZoneProps extends Omit<
  Select.ButtonsProps<TimeZone>,
  "keys"
> {}

export const SelectTimeZone = (props: SelectTimeZoneProps): ReactElement => (
  <Select.Buttons {...props} keys={TIME_ZONES}>
    <Select.Button itemKey="UTC" tooltip="UTC">
      UTC
    </Select.Button>
    <Select.Button itemKey="local" tooltip="Local time zone">
      Local
    </Select.Button>
  </Select.Buttons>
);
