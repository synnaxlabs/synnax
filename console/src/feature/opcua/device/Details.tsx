// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/feature/opcua/device/Details.css";

import { status, TimeStamp } from "@synnaxlabs/client";
import { Button } from "@synnaxlabs/lyra/button";
import { Caret } from "@synnaxlabs/lyra/caret";
import { Description } from "@synnaxlabs/lyra/description";
import { Flex } from "@synnaxlabs/lyra/flex";
import { Haul } from "@synnaxlabs/lyra/haul";
import { Header } from "@synnaxlabs/lyra/header";
import { useAsyncEffect, useSyncedRef } from "@synnaxlabs/lyra/hooks";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Status } from "@synnaxlabs/lyra/status";
import { Text } from "@synnaxlabs/lyra/text";
import { Synnax } from "@synnaxlabs/pluto";
import { math } from "@synnaxlabs/x";
import { type ReactElement, type ReactNode, useRef, useState } from "react";

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
  /** The name of the node, when the Browser knows it. */
  name?: string;
  collapsed: boolean;
  onCollapsedChange: (collapsed: boolean) => void;
}

interface State {
  /** The node and field that the state answers. */
  node: string;
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

const type = ({ typeName, valueRank, arrayDimensions }: NodeDetails): string => {
  const name = typeName ?? "Unknown";
  if (valueRank == null || valueRank === SCALAR_RANK) return name;
  if (valueRank < 0) return `${name}, scalar or array`;
  if (arrayDimensions == null || arrayDimensions.length === 0) return `${name} array`;
  return `${name} array [${arrayDimensions.join(" × ")}]`;
};

// OPC UA reserves 0 for continuous sampling and -1 for an unknown interval, and neither
// limits the sample rate.
const sampling = (interval?: number): string | undefined =>
  interval != null && interval > 0 ? `${interval} ms minimum` : undefined;

const statusVariant = (value: string): status.Variant => {
  if (value.startsWith("Bad")) return "error";
  if (value.startsWith("Uncertain")) return "warning";
  return "success";
};

const isRecord = (value: unknown): value is Record<string, unknown> =>
  typeof value === "object" && value != null && !Array.isArray(value);

const PREVIEW_LENGTH = 8;

const text = (value: unknown): string => {
  if (typeof value === "number" && !Number.isInteger(value))
    return math.smartRound(value).toString();
  if (!Array.isArray(value)) return JSON.stringify(value);
  const shown = value.slice(0, PREVIEW_LENGTH).map(text);
  if (shown.length < value.length) shown.push("…");
  return `[${shown.join(", ")}]`;
};

// A structure lists its fields by path, as the Browser names its members.
const fields = (value: Record<string, unknown>, path = ""): Array<[string, string]> =>
  Object.entries(value).flatMap(([key, field]) => {
    const fieldPath = path === "" ? key : `${path}.${key}`;
    return isRecord(field) ? fields(field, fieldPath) : [[fieldPath, text(field)]];
  });

interface Preview {
  text?: string;
  fields?: Array<[string, string]>;
  /** The length of an array that text shows only the start of. */
  total?: number;
}

// The Driver prints every value as compact JSON, and null when the server sent none.
const preview = (value: string): Preview | undefined => {
  const parsed: unknown = JSON.parse(value);
  if (parsed == null) return undefined;
  if (isRecord(parsed)) return { fields: fields(parsed) };
  const cut = Array.isArray(parsed) && parsed.length > PREVIEW_LENGTH;
  return { text: text(parsed), total: cut ? parsed.length : undefined };
};

const Reading = ({
  value,
  status,
  sourceTimestamp,
  serverTimestamp,
}: NodeDetails): ReactElement => {
  const shown = value == null ? undefined : preview(value);
  const time = sourceTimestamp ?? serverTimestamp;
  return (
    <Flex.Box y gap="small">
      {shown?.text != null && (
        <Text.Text variant="code" color={11} className={CSS.BE("opc-details", "value")}>
          {shown.text}
        </Text.Text>
      )}
      {shown?.fields != null && (
        <Description.List>
          {shown.fields.map(([path, value]) => (
            <Description.Item key={path}>
              <Description.Label>{path}</Description.Label>
              <Description.Value
                variant="code"
                color={11}
                className={CSS.BE("opc-details", "value")}
              >
                {value}
              </Description.Value>
            </Description.Item>
          ))}
        </Description.List>
      )}
      {shown?.total != null && (
        <Text.Text level="small" color={9}>
          {`Showing ${PREVIEW_LENGTH} of ${shown.total.toLocaleString()} values`}
        </Text.Text>
      )}
      {status != null && (
        <Flex.Box x wrap align="center" justify="between" gap="medium">
          <Status.Summary
            level="small"
            variant={statusVariant(status)}
            message={status}
          />
          {time != null && (
            <Text.Text
              level="small"
              color={9}
              className={CSS.BE("opc-details", "time")}
            >
              {new TimeStamp(time).toString("dateTime", "local")}
            </Text.Text>
          )}
        </Flex.Box>
      )}
    </Flex.Box>
  );
};

// A node ID wraps after its separators, not inside a name.
const breakAfterSeparators = (id: string): ReactNode[] =>
  id
    .split(/(?<=[;.:/])/)
    .flatMap((part, i) => (i === 0 ? [part] : [<wbr key={i} />, part]));

interface RowProps extends Description.ValueProps {
  label: string;
  value?: string;
}

const Row = ({ label, value, children, ...rest }: RowProps): ReactElement | null => {
  if (value == null || value === "") return null;
  return (
    <Description.Item>
      <Description.Label>{label}</Description.Label>
      <Description.Value {...rest}>{children ?? value}</Description.Value>
    </Description.Item>
  );
};

/** Shows the attributes and the current value of one node of a server. */
export const Details = ({
  device,
  nodeId,
  field,
  name,
  collapsed,
  onCollapsedChange,
}: DetailsProps): ReactElement => {
  const client = Synnax.use();
  const { startDrag } = Haul.useDrag({ type: HAUL_TYPE });
  const deviceRef = useSyncedRef(device);
  const [state, setState] = useState<State | null>(null);
  const [reads, setReads] = useState(0);
  const node = JSON.stringify([nodeId, field]);
  const request = JSON.stringify([nodeId, field, reads]);
  const reading = useRef<Promise<unknown>>(Promise.resolve());
  useAsyncEffect(
    async (signal) => {
      if (client == null || collapsed) return;
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
        next = { node, request, details: await read };
      } catch (e) {
        next = {
          node,
          request,
          failure: status.fromException(e, "Failed to read node"),
        };
      }
      if (!signal.aborted) setState(next);
    },
    [client, deviceRef, node, request, collapsed],
  );
  // The last answer stays in view while the next read runs, so the pane never empties.
  const fresh = state?.node === node;
  const loading = !collapsed && state?.request !== request;
  const details = state?.details;
  const failure = state?.failure;
  let content: ReactElement | null = null;
  if (failure != null)
    content = <Status.Summary level="small" status={failure} center={false} />;
  else if (details != null)
    content = (
      <>
        {details.nodeClass === "Variable" && <Reading {...details} />}
        <Description.List level="small">
          <Row
            label="Type"
            value={details.nodeClass === "Variable" ? type(details) : undefined}
          />
          <Row
            label="Class"
            value={details.nodeClass === "Variable" ? undefined : details.nodeClass}
          />
          <Row
            label="Access"
            value={details.nodeClass === "Variable" ? access(details) : undefined}
          />
          <Row label="Sampling" value={sampling(details.minimumSamplingInterval)} />
          <Row label="Node ID" value={details.node.nodeId} variant="code">
            {breakAfterSeparators(details.node.nodeId)}
          </Row>
          <Row label="Field" value={details.node.field} variant="code">
            {breakAfterSeparators(details.node.field)}
          </Row>
          <Row
            label="Description"
            value={
              details.description === details.node.name ||
              details.description === details.displayName
                ? undefined
                : details.description
            }
          />
        </Description.List>
      </>
    );
  const draggable =
    fresh && (details?.nodeClass === "Variable" || details?.nodeClass === "Object");
  const handleDragStart = () => {
    if (draggable && details != null)
      startDrag([createHaulItem({ ...details.node, key: nodeIdentity(details.node) })]);
  };
  return (
    <Flex.Box y empty className={CSS.B("opc-details")}>
      <Header.Header level="small">
        <Header.ButtonTitle
          onClick={() => onCollapsedChange(!collapsed)}
          aria-expanded={!collapsed}
          draggable={draggable}
          onDragStart={handleDragStart}
        >
          <Caret.Animated
            enabled={!collapsed}
            enabledLoc="bottom"
            disabledLoc="right"
          />
          {name ?? (fresh ? details?.node.name : undefined) ?? nodeId}
        </Header.ButtonTitle>
        {!collapsed && (
          <Header.Actions>
            {loading && <Status.Loading grow={false} />}
            <Button.Button
              onClick={() => setReads((r) => r + 1)}
              size="small"
              variant="text"
              tooltip="Read again"
            >
              <Icon.Refresh />
            </Button.Button>
          </Header.Actions>
        )}
      </Header.Header>
      {!collapsed && content != null && (
        <Flex.Box
          y
          gap={3}
          className={CSS.cls(
            CSS.BE("opc-details", "body"),
            !fresh && CSS.BEM("opc-details", "body", "stale"),
          )}
        >
          {content}
        </Flex.Box>
      )}
    </Flex.Box>
  );
};
