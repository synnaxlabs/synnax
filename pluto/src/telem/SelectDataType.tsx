// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Select } from "@synnaxlabs/lyra/select";
import { DataType } from "@synnaxlabs/x";
import type { ReactElement } from "react";

import { resolveDataTypeIcon } from "@/telem/resolveDataTypeIcon";
import { resolveDataTypeName } from "@/telem/resolveDataTypeName";

const resolveIcon = (d: DataType) => {
  const Resolved = resolveDataTypeIcon(d);
  return Resolved != null ? <Resolved /> : undefined;
};

const DATA = DataType.ALL.filter((d) => d !== DataType.UNKNOWN).map((d) => ({
  key: d.toString(),
  name: resolveDataTypeName(d),
  icon: resolveIcon(d),
}));

const FIXED_DENSITY_DATA = DATA.filter((d) => !new DataType(d.key).isVariable);

export interface SelectDataTypeProps extends Omit<
  Select.SingleSimpleProps<string>,
  "children" | "resourceName"
> {
  hideVariableDensity?: boolean;
  hideDataTypes?: DataType[];
}

export const SelectDataType = ({
  hideVariableDensity = false,
  hideDataTypes = DEFAULT_HIDDEN_DATA_TYPES,
  ...rest
}: SelectDataTypeProps): ReactElement => {
  const data = hideVariableDensity ? FIXED_DENSITY_DATA : DATA;
  const filteredData = data.filter((d) => !hideDataTypes.some((h) => h.equals(d.key)));
  return (
    <Select.Simple<string> {...rest} resourceName="data type">
      {filteredData.map(({ key, name, icon }) => (
        <Select.Item key={key} itemKey={key}>
          {icon}
          {name}
        </Select.Item>
      ))}
    </Select.Simple>
  );
};

const DEFAULT_HIDDEN_DATA_TYPES: DataType[] = [];
