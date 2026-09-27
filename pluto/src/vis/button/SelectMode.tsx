// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Select } from "@synnaxlabs/lyra/select";
import { type ReactElement } from "react";

import { type Mode } from "@/vis/button/use";

interface SelectButtonModeProps extends Select.ButtonsProps<Mode> {}

export const SelectMode = (props: SelectButtonModeProps): ReactElement => (
  <Select.Buttons {...props}>
    <Select.Item itemKey="fire" tooltip="Output true when clicked">
      Fire
    </Select.Item>
    <Select.Item itemKey="momentary" tooltip="Output true on press, false on release">
      Momentary
    </Select.Item>
    <Select.Item itemKey="pulse" tooltip="Output true, then false, on click">
      Pulse
    </Select.Item>
  </Select.Buttons>
);
