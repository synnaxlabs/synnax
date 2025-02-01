// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type List, Select, Text } from "@synnaxlabs/pluto";
import { type ReactElement, useMemo } from "react";

import { type ScannedNode } from "@/hardware/opc/device/types";
import { type NodeId, parseNodeId } from "@/hardware/opc/task/types";

interface NodeEntry extends NodeId {
  name: string;
  key: string;
  dataType: string;
  isArray: boolean;
}

const SELECT_NODE_COLUMNS: Array<List.ColumnSpec<string, NodeEntry>> = [
  { name: "Name", key: "name" },
  { name: "Identifier", key: "identifier" },
  { name: "Namespace", key: "namespaceIndex" },
  { name: "Data Type", key: "dataType" },
  {
    name: "Is Array",
    key: "isArray",
    width: 100,
    render: ({ entry: { isArray } }) => (
      <Text.Text level="p">{isArray ? "Yes" : "No"}</Text.Text>
    ),
  },
];

interface SelectNodeProps extends Omit<Select.SingleProps<string, NodeEntry>, "data"> {
  data: ScannedNode[];
}

export const SelectNode = ({ data, ...props }: SelectNodeProps): ReactElement => {
  const transformedData = useMemo(
    () =>
      data
        .map((c) => {
          const n = parseNodeId(c.nodeId);
          if (n == null) return null;
          return {
            name: c.name,
            key: c.nodeId,
            ...n,
            dataType: c.dataType,
            isArray: c.isArray,
          };
        })
        .filter((n) => n != null),
    [data],
  );
  return (
    <Select.Single<string, NodeEntry>
      {...props}
      columns={SELECT_NODE_COLUMNS}
      data={transformedData}
      entryRenderKey={(e) => `${e.name} (${e.key})`}
    />
  );
};
