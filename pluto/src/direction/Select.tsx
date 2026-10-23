// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Icon } from "@synnaxlabs/lyra/icon";
import { Select as BaseSelect } from "@synnaxlabs/lyra/select";
import { type direction } from "@synnaxlabs/x";
import { type ReactElement } from "react";

export interface SelectProps extends BaseSelect.ButtonsProps<direction.Direction> {
  yDirection?: "up" | "down";
}

export const Select = ({ yDirection = "up", ...rest }: SelectProps): ReactElement => (
  <BaseSelect.Buttons {...rest}>
    <BaseSelect.Item itemKey="x">
      <Icon.Arrow.Right />
    </BaseSelect.Item>
    <BaseSelect.Item itemKey="y">
      {yDirection === "up" ? <Icon.Arrow.Up /> : <Icon.Arrow.Down />}
    </BaseSelect.Item>
  </BaseSelect.Buttons>
);
