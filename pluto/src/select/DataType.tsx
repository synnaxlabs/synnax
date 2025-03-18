// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { caseconv, DataType as TelemDataType } from "@synnaxlabs/x";
import { type ReactElement } from "react";

import { type List } from "@/list";
import { DropdownButton, type DropdownButtonProps } from "@/select/Button";

interface ListEntry {
  key: string;
  name: string;
}

const ALLCAPS = new Set([TelemDataType.UUID, TelemDataType.JSON]);

const DATA: ListEntry[] = TelemDataType.ALL.filter(
  (d) => d !== TelemDataType.UNKNOWN,
).map((d) => ({
  key: d.toString(),
  name: ALLCAPS.has(d) ? d.toString().toUpperCase() : caseconv.capitalize(d.toString()),
}));

const FIXED_DENSITY_DATA = DATA.filter((d) => !new TelemDataType(d.key).isVariable);

const COLUMNS: Array<List.ColumnSpec<string, ListEntry>> = [
  { key: "name", name: "Name" },
];

export interface DataTypeProps
  extends Omit<
    DropdownButtonProps<string, ListEntry>,
    "data" | "columns" | "entryRenderKey"
  > {
  hideVariableDensity?: boolean;
}

export const DataType = ({
  hideVariableDensity = false,
  ...rest
}: DataTypeProps): ReactElement => (
  <DropdownButton<string, ListEntry>
    {...rest}
    data={hideVariableDensity ? FIXED_DENSITY_DATA : DATA}
    columns={COLUMNS}
    entryRenderKey="name"
  />
);
