// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ReactElement } from "react";

import { type List } from "@/list";
import { Select as Core } from "@/select";
import { type Status } from "@/status";
import { Text } from "@/status/Text";

interface ListEntry {
  key: Status.Variant;
  name: string;
}

const OPTIONS: ListEntry[] = [
  {
    key: "success",
    name: "Success",
  },
  {
    key: "error",
    name: "Error",
  },
  {
    key: "warning",
    name: "Warning",
  },
  {
    key: "info",
    name: "Info",
  },
];

const COLUMNS: Array<List.ColumnSpec<string, ListEntry>> = [
  {
    key: "name",
    name: "Name",
    render: ({ entry: { key, name } }) => <Text variant={key}>{name}</Text>,
  },
];

export interface SelectProps
  extends Omit<
    Core.DropdownButtonProps<string, ListEntry>,
    "data" | "columns" | "entryRenderKey"
  > {}

export const Select = (props: SelectProps): ReactElement => (
  <Core.DropdownButton
    data={OPTIONS}
    columns={COLUMNS}
    entryRenderKey="name"
    {...props}
  >
    {(p) => (
      <Core.BaseButton {...p}>
        <Text variant={p.selected?.key ?? "info"}>{p.selected?.name}</Text>
      </Core.BaseButton>
    )}
  </Core.DropdownButton>
);
