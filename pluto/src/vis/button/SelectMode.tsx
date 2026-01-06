// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ReactElement } from "react";

import { Select } from "@/select";
import { type Mode, MODES } from "@/vis/button/use";

interface SelectButtonModeProps extends Omit<Select.ButtonsProps<Mode>, "keys"> {}

export const SelectMode = (props: SelectButtonModeProps): ReactElement => (
  <Select.Buttons {...props} keys={MODES}>
    <Select.Button itemKey="fire" tooltip="Output true when clicked">
      Fire
    </Select.Button>
    <Select.Button itemKey="momentary" tooltip="Output true on press, false on release">
      Momentary
    </Select.Button>
    <Select.Button
      itemKey="pulse"
      tooltip="Output true and then immediately output false on click"
    >
      Pulse
    </Select.Button>
  </Select.Buttons>
);
