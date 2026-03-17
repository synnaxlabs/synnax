// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { caseconv, DataType } from "@synnaxlabs/x";
import { type ReactElement, useMemo } from "react";

import { type Select } from "@/select";
import { Static as SelectStatic } from "@/select/Static";
import { resolveDataTypeIcon } from "@/telem/resolveDataTypeIcon";

const ALL_CAPS = new Set([DataType.UUID, DataType.JSON]);

const resolveIcon = (d: DataType) => {
  const Resolved = resolveDataTypeIcon(d);
  return Resolved != null ? <Resolved /> : undefined;
};

const DATA: Select.StaticEntry<string>[] = DataType.ALL.filter(
  (d) => d !== DataType.UNKNOWN,
).map((d) => ({
  key: d.toString(),
  name: ALL_CAPS.has(d)
    ? d.toString().toUpperCase()
    : d.isNumeric && d !== DataType.TIMESTAMP
      ? d.toString()
      : caseconv.capitalize(d.toString()),
  icon: resolveIcon(d),
}));

const FIXED_DENSITY_DATA = DATA.filter((d) => !new DataType(d.key).isVariable);

export interface SelectDataTypeProps extends Omit<
  Select.StaticProps<string>,
  "data" | "resourceName"
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
  const filteredData = useMemo(
    () => data.filter((d) => !hideDataTypes.some((h) => h.equals(d.key))),
    [hideDataTypes, data],
  );
  return <SelectStatic {...rest} data={filteredData} resourceName="data type" />;
};

const DEFAULT_HIDDEN_DATA_TYPES: DataType[] = [];
