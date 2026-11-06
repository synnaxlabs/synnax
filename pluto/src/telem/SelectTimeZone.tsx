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

export interface SelectTimeZoneProps extends Select.ButtonsProps<TimeZone> {
  disabled?: boolean;
}

export const SelectTimeZone = ({
  disabled,
  ...rest
}: SelectTimeZoneProps): ReactElement => (
  <Select.Buttons {...rest}>
    <Select.Item itemKey="UTC" tooltip="UTC" disabled={disabled}>
      UTC
    </Select.Item>
    <Select.Item itemKey="local" tooltip="Local time zone" disabled={disabled}>
      Local
    </Select.Item>
  </Select.Buttons>
);
