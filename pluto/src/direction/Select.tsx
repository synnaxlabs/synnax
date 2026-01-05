// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { direction } from "@synnaxlabs/x";
import { type ReactElement } from "react";

import { Icon } from "@/icon";
import { Select as CoreSelect } from "@/select";

export interface SelectProps extends Omit<
  CoreSelect.ButtonsProps<direction.Direction>,
  "keys"
> {
  yDirection?: "up" | "down";
}

export const Select = ({ yDirection = "up", ...rest }: SelectProps): ReactElement => (
  <CoreSelect.Buttons {...rest} keys={direction.DIRECTIONS}>
    <CoreSelect.Button itemKey="x">
      <Icon.Arrow.Right />
    </CoreSelect.Button>
    <CoreSelect.Button itemKey="y">
      {yDirection === "up" ? <Icon.Arrow.Up /> : <Icon.Arrow.Down />}
    </CoreSelect.Button>
  </CoreSelect.Buttons>
);
