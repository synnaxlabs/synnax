import { useMemo, type ReactElement } from "react";

import { Form, type List, Select, Synnax } from "@synnaxlabs/pluto";
import { useQuery } from "@tanstack/react-query";

import {
  type DeviceNodeProperties,
  parseNodeId,
  type DeviceProperties,
  type NodeId,
} from "@/hardware/opc/types";

interface NodeEntry extends NodeId {
  name: string;
  key: string;
  dataType: string;
}

const SELECT_NODE_COLUMNS: Array<List.ColumnSpec<string, NodeEntry>> = [
  {
    name: "Name",
    key: "name",
  },
  {
    name: "Identifier",
    key: "identifier",
  },
  {
    name: "Namespace",
    key: "namespaceIndex",
  },
  {
    name: "Data Type",
    key: "dataType",
  },
];

interface SelectNodeProps
  extends Omit<Select.SingleProps<string, NodeEntry>, "columns" | "data"> {
  data: DeviceNodeProperties[];
}

export const SelectNode = ({ data, ...props }: SelectNodeProps): ReactElement => {
  const transformedData = useMemo(
    () =>
      data
        .map((c) => {
          const n = parseNodeId(c.nodeId);
          if (n == null) return null;
          return { name: c.name, key: c.nodeId, ...n, dataType: c.dataType };
        })
        .filter((n) => n != null) as NodeEntry[],
    [data],
  );
  console.log(transformedData);
  return (
    <Select.Single<string, NodeEntry>
      columns={SELECT_NODE_COLUMNS}
      data={transformedData}
      entryRenderKey={(e) => `${e.name} (${e.key})`}
      {...props}
    />
  );
};

export interface SelectNodeRemoteProps extends Omit<SelectNodeProps, "data"> {
  device: string;
}

export const SelectNodeRemote = ({
  device,
  ...props
}: SelectNodeRemoteProps): ReactElement => {
  const client = Synnax.use();
  const nodes = useQuery({
    queryKey: [client?.key, device],
    queryFn: async () => {
      if (client == null) return;
      const d = await client.hardware.devices.retrieve<DeviceProperties>([device]);
      if (d.length === 0) return [];
      return d[0].properties.channels;
    },
  });
  return <SelectNode data={nodes.data ?? []} {...props} />;
};
