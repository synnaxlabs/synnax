// Copyright 2025 Synnax Labs, Inc.
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

export interface SelectProps extends CoreSelect.SingleProps<direction.Direction> {
  yDirection?: "up" | "down";
}

const DATA: direction.Direction[] = [...direction.DIRECTIONS];

export const Select = ({
  yDirection = "up",
  value,
  onChange,
  ...rest
}: SelectProps): ReactElement => {
  const { onSelect, ...selectProps } = CoreSelect.useSingle({
    value,
    onChange,
    data: DATA,
  });
  return (
    <CoreSelect.Buttons {...rest} {...selectProps} value={value} onSelect={onSelect}>
      <CoreSelect.ButtonIcon itemKey="x">
        <Icon.Arrow.Right />
      </CoreSelect.ButtonIcon>
      <CoreSelect.ButtonIcon itemKey="y">
        {yDirection === "up" ? <Icon.Arrow.Up /> : <Icon.Arrow.Down />}
      </CoreSelect.ButtonIcon>
    </CoreSelect.Buttons>
  );
};
