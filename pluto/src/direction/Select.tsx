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
import { Select as BaseSelect } from "@/select";

export interface SelectProps extends Omit<
  BaseSelect.ButtonsProps<direction.Direction>,
  "keys"
> {
  yDirection?: "up" | "down";
}

export const Select = ({ yDirection = "up", ...rest }: SelectProps): ReactElement => (
  <BaseSelect.Buttons {...rest} keys={direction.DIRECTIONS}>
    <BaseSelect.Button itemKey="x">
      <Icon.Arrow.Right />
    </BaseSelect.Button>
    <BaseSelect.Button itemKey="y">
      {yDirection === "up" ? <Icon.Arrow.Up /> : <Icon.Arrow.Down />}
    </BaseSelect.Button>
  </BaseSelect.Buttons>
);
