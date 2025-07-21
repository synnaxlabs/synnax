// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { caseconv, DataType } from "@synnaxlabs/x";
import { type ReactElement } from "react";

import { Select } from "@/select";

const ALL_CAPS = new Set([DataType.UUID, DataType.JSON]);

const DATA: Select.SimplyEntry<string>[] = DataType.ALL.filter(
  (d) => d !== DataType.UNKNOWN,
).map((d) => ({
  key: d.toString(),
  name: ALL_CAPS.has(d)
    ? d.toString().toUpperCase()
    : caseconv.capitalize(d.toString()),
}));

const FIXED_DENSITY_DATA = DATA.filter((d) => !new DataType(d.key).isVariable);

export interface SelectDataTypeProps
  extends Omit<Select.SimpleProps<string>, "data" | "resourceName"> {
  hideVariableDensity?: boolean;
}

export const SelectDataType = ({
  hideVariableDensity = false,
  ...rest
}: SelectDataTypeProps): ReactElement => {
  const data = hideVariableDensity ? FIXED_DENSITY_DATA : DATA;
  return <Select.Simple {...rest} data={data} resourceName="Data Type" />;
};
