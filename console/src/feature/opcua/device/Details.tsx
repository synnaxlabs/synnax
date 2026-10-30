// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/feature/opcua/device/Details.css";

import { status } from "@synnaxlabs/client";
import { Button } from "@synnaxlabs/lyra/button";
import { Flex } from "@synnaxlabs/lyra/flex";
import { Haul } from "@synnaxlabs/lyra/haul";
import { Header } from "@synnaxlabs/lyra/header";
import { useAsyncEffect, useSyncedRef } from "@synnaxlabs/lyra/hooks";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Status } from "@synnaxlabs/lyra/status";
import { Text } from "@synnaxlabs/lyra/text";
import { Synnax } from "@synnaxlabs/pluto";
import { type ReactElement, useRef, useState } from "react";

import { commandTimeout } from "@/feature/opcua/device/commandTimeout";
import { createHaulItem, HAUL_TYPE } from "@/feature/opcua/device/haul";
import { executeScanCommand } from "@/feature/opcua/device/scanCommand";
import { type Device } from "@/feature/opcua/device/types";
import {
  type NodeDetails,
  nodeDetailsZ,
  nodeIdentity,
  READ_NODE_COMMAND_TYPE,
} from "@/feature/opcua/task/types";
import { CSS } from "@/platform/css";

interface DetailsProps {
  device: Device;
  nodeId: string;
  field: string;
  /** The node ID of the node that the browser found this one under. */
  parent?: string;
}

interface State {
  /** The node, field, and read count that the state answers. */
  request: string;
  details?: NodeDetails;
  failure?: status.Status;
}

const access = ({ readable, writable, historyReadable }: NodeDetails): string => {
  const kinds: string[] = [];
  if (readable === true) kinds.push("Read");
  if (writable === true) kinds.push("Write");
  if (historyReadable === true) kinds.push("History");
  return kinds.length === 0 ? "None" : kinds.join(", ");
};

const SCALAR_RANK = -1;

const shape = ({ valueRank, arrayDimensions }: NodeDetails): string | undefined => {
  if (valueRank == null) return undefined;
  if (valueRank === SCALAR_RANK) return "Scalar";
  if (valueRank < 0) return "Scalar or array";
  if (arrayDimensions == null || arrayDimensions.length === 0) return "Array";
  return `Array [${arrayDimensions.join(" × ")}]`;
};

interface RowProps {
  label: string;
  value?: string;
}

const Row = ({ label, value }: RowProps): ReactElement | null => {
  if (value == null || value === "") return null;
  return (
    <Flex.Box x gap="small" className={CSS.BE("opc-details", "row")}>
      <Text.Text level="small" color={8} className={CSS.BE("opc-details", "label")}>
        {label}
      </Text.Text>
      <Text.Text level="small" color={10} className={CSS.BE("opc-details", "value")}>
        {value}
      </Text.Text>
    </Flex.Box>
  );
};

/** Shows the attributes and the current value of one node of a server. */
export const Details = ({
  device,
  nodeId,
  field,
  parent,
}: DetailsProps): ReactElement => {
  const client = Synnax.use();
  const { startDrag } = Haul.useDrag({ type: HAUL_TYPE });
  const deviceRef = useSyncedRef(device);
  const [state, setState] = useState<State | null>(null);
  const [reads, setReads] = useState(0);
  const request = JSON.stringify([nodeId, field, reads]);
  const reading = useRef<Promise<unknown>>(Promise.resolve());
  useAsyncEffect(
    async (signal) => {
      if (client == null) return;
      // The Driver runs one scan command at a time, so a read waits for the open one,
      // and a selection that changes meanwhile is never read.
      await reading.current;
      if (signal.aborted) return;
      const { rack, properties } = deviceRef.current;
      const read = executeScanCommand(client, rack, {
        type: READ_NODE_COMMAND_TYPE,
        args: { connection: properties.connection, node_id: nodeId, field },
        schema: nodeDetailsZ,
        timeout: commandTimeout(properties.connection),
      });
      reading.current = read.catch(() => undefined);
      let next: State;
      try {
        next = { request, details: await read };
      } catch (e) {
        next = { request, failure: status.fromException(e, "Failed to read node") };
      }
      if (!signal.aborted) setState(next);
    },
    [client, deviceRef, request],
  );
  const current = state?.request === request ? state : null;
  const details = current?.details;
  const failure = current?.failure;
  let content: ReactElement | null = null;
  if (failure != null) content = <Status.Summary status={failure} />;
  else if (details != null)
    content = (
      <Flex.Box y gap="tiny" className={CSS.BE("opc-details", "rows")}>
        <Row label="Node ID" value={details.node.nodeId} />
        <Row label="Field" value={details.node.field} />
        <Row label="Class" value={details.nodeClass} />
        <Row label="Description" value={details.description} />
        <Row label="Data type" value={details.typeName} />
        <Row label="Shape" value={shape(details)} />
        <Row
          label="Access"
          value={details.nodeClass === "Variable" ? access(details) : undefined}
        />
        <Row
          label="Historizing"
          value={details.historizing == null ? undefined : String(details.historizing)}
        />
        <Row
          label="Sampling"
          value={
            details.minimumSamplingInterval == null
              ? undefined
              : `${details.minimumSamplingInterval} ms minimum`
          }
        />
        <Row
          label="Arguments"
          value={details.inputArguments
            ?.map(({ name, typeName }) => `${name} (${typeName})`)
            .join(", ")}
        />
        <Row label="Value" value={details.value} />
        <Row label="Status" value={details.status} />
        <Row label="Source time" value={details.sourceTimestamp} />
        <Row label="Server time" value={details.serverTimestamp} />
      </Flex.Box>
    );
  const draggable =
    details?.nodeClass === "Variable" ||
    details?.nodeClass === "Object" ||
    details?.nodeClass === "ObjectType" ||
    details?.nodeClass === "Method";
  const handleDragStart = () => {
    if (details != null)
      startDrag([
        createHaulItem({ ...details.node, key: nodeIdentity(details.node), parent }),
      ]);
  };
  return (
    <Flex.Box y empty className={CSS.B("opc-details")}>
      <Header.Header>
        <Header.Title
          weight={500}
          color={10}
          level="small"
          draggable={draggable}
          onDragStart={handleDragStart}
        >
          {details?.node.name ?? (failure == null ? "Reading node" : "Node")}
        </Header.Title>
        <Header.Actions>
          <Button.Button
            onClick={() => setReads((r) => r + 1)}
            size="small"
            variant="text"
            tooltip="Read again"
          >
            <Icon.Refresh />
          </Button.Button>
        </Header.Actions>
      </Header.Header>
      {content}
    </Flex.Box>
  );
};
