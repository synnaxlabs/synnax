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

import { Icon } from "@/icon";
import { type Select } from "@/select";
import { Static as SelectStatic } from "@/select/Static";

const ALL_CAPS = new Set([DataType.UUID, DataType.JSON]);

const resolveIcon = (d: DataType) => {
  if (d.equals(DataType.JSON)) return <Icon.JSON />;
  if (d.isInteger) return <Icon.Binary />;
  if (d.isFloat) return <Icon.Decimal />;
  if (d.equals(DataType.STRING) || d.equals(DataType.UUID)) return <Icon.String />;
  if (d.equals(DataType.TIMESTAMP)) return <Icon.Time />;
  return undefined;
};

const DATA: Select.StaticEntry<string>[] = DataType.ALL.filter(
  (d) => d !== DataType.UNKNOWN,
).map((d) => ({
  key: d.toString(),
  name: ALL_CAPS.has(d)
    ? d.toString().toUpperCase()
    : caseconv.capitalize(d.toString()),
  icon: resolveIcon(d),
}));

const FIXED_DENSITY_DATA = DATA.filter((d) => !new DataType(d.key).isVariable);

export interface SelectDataTypeProps
  extends Omit<Select.StaticProps<string>, "data" | "resourceName"> {
  hideVariableDensity?: boolean;
}

export const SelectDataType = ({
  hideVariableDensity = false,
  ...rest
}: SelectDataTypeProps): ReactElement => {
  const data = hideVariableDensity ? FIXED_DENSITY_DATA : DATA;
  return <SelectStatic {...rest} data={data} resourceName="data type" />;
};
