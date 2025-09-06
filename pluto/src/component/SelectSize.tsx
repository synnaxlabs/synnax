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

export interface SelectComponentSizeProps
  extends Omit<Select.ButtonsProps<Size>, "keys"> {}

export const SelectSize = (props: SelectComponentSizeProps): ReactElement => (
  <Select.Buttons {...props} keys={SIZES}>
    <Select.Button itemKey="tiny">S</Select.Button>
    <Select.Button itemKey="small">M</Select.Button>
    <Select.Button itemKey="medium">L</Select.Button>
  </Select.Buttons>
);
