// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type label } from "@synnaxlabs/client";
import { Icon } from "@synnaxlabs/media";
import { type ReactElement } from "react";

import { type List } from "@/list";
import { Select } from "@/select";
import { Synnax } from "@/synnax";
import { Tag } from "@/tag";

const labelColumns: Array<List.ColumnSpec<label.Key, label.Label>> = [
  {
    key: "color",
    name: "Color",
    width: 50,
    render: (v) => (
      <Icon.Circle
        style={{ fontSize: 10, marginLeft: 10, marginRight: 30 }}
        color={v.entry.color}
      />
    ),
  },
  { key: "name", name: "Name" },
];

export interface SelectLabelsProps
  extends Omit<Select.MultipleProps<label.Key, label.Label>, "columns" | "searcher"> {}

export const SelectLabels = (props: SelectLabelsProps): ReactElement => {
  const client = Synnax.use();
  return (
    <Select.Multiple
      {...props}
      searcher={client?.labels}
      columns={labelColumns}
      renderTag={(v) => {
        if (v.entry == null) return <Tag.Tag color="gray">{v.entryKey}</Tag.Tag>;
        return <Tag.Tag color={v.entry.color}>{v.entry.name}</Tag.Tag>;
      }}
    />
  );
};
