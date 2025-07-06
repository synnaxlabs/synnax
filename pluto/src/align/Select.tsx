// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ReactElement } from "react";

import { type Alignment, ALIGNMENTS } from "@/align/Space";
import { Icon } from "@/icon";
import { Select as CoreSelect } from "@/select";

const DATA: Alignment[] = [...ALIGNMENTS];

export interface SelectProps extends CoreSelect.SingleProps<Alignment> {}

export const Select = ({ onChange, value, ...rest }: SelectProps): ReactElement => {
  const { onSelect, ...selectProps } = CoreSelect.useSingle({
    value,
    onChange,
    data: DATA,
  });
  return (
    <CoreSelect.Buttons {...rest} {...selectProps} value={value} onSelect={onSelect}>
      <CoreSelect.ButtonIcon itemKey="start">
        <Icon.TextAlign.Left />
      </CoreSelect.ButtonIcon>
      <CoreSelect.ButtonIcon itemKey="center">
        <Icon.TextAlign.Center />
      </CoreSelect.ButtonIcon>
      <CoreSelect.ButtonIcon itemKey="end">
        <Icon.TextAlign.Right />
      </CoreSelect.ButtonIcon>
    </CoreSelect.Buttons>
  );
};
