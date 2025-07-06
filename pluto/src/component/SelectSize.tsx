// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ReactElement } from "react";

import { type Size, SIZES } from "@/component/size";
import { Select } from "@/select";

const SIZE_DATA = [...SIZES];

export interface SelectComponentSizeProps extends Select.SingleProps<Size> {}

export const SelectSize = ({
  value,
  onChange,
  ...rest
}: SelectComponentSizeProps): ReactElement => {
  const { onSelect, ...selectProps } = Select.useSingle({
    data: SIZE_DATA,
    value,
    onChange,
    ...rest,
  });
  return (
    <Select.Buttons {...rest} {...selectProps} onSelect={onSelect} value={value}>
      <Select.ButtonIcon itemKey="tiny">L</Select.ButtonIcon>
      <Select.ButtonIcon itemKey="small">M</Select.ButtonIcon>
      <Select.ButtonIcon itemKey="medium">L</Select.ButtonIcon>
    </Select.Buttons>
  );
};
