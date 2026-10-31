// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/feature/opcua/task/Task.css";

import { channel, NotFoundError, type Synnax } from "@synnaxlabs/client";
import { Button } from "@synnaxlabs/lyra/button";
import { Flex } from "@synnaxlabs/lyra/flex";
import { Form as PForm } from "@synnaxlabs/lyra/form";
import { Haul } from "@synnaxlabs/lyra/haul";
import { Header } from "@synnaxlabs/lyra/header";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Text } from "@synnaxlabs/lyra/text";
import { DataType, errors, id } from "@synnaxlabs/x";
import { type FC, type ReactElement, type ReactNode } from "react";

import { Browser } from "@/feature/opcua/device/Browser";
import { HAUL_TYPE, type HaulItem, isHaulItem } from "@/feature/opcua/device/haul";
import { use } from "@/feature/opcua/device/queries";
import { Select } from "@/feature/opcua/device/Select";
import { type Device, SCHEMAS } from "@/feature/opcua/device/types";
import { useConnectModal } from "@/feature/opcua/device/useConnectModal";
import { findFreeName } from "@/feature/opcua/task/findFreeName";
import {
  deployEventConfigZ,
  EVENT_SCHEMAS,
  EVENT_TYPE,
  type EventField,
  type EventSchemas,
  type ScannedNode,
} from "@/feature/opcua/task/types";
import { CSS } from "@/platform/css";
import { Device as PlatformDevice } from "@/platform/device";
import { Selector } from "@/platform/selector";
import { Task } from "@/platform/task";

const Properties = (): ReactElement => (
  <>
    <Select />
    <Flex.Box x>
      <Task.Fields.DataSaving />
      <Task.Fields.AutoStart />
    </Flex.Box>
  </>
);

interface NodeListProps {
  path: string;
  title: string;
  /** The class of the nodes that the list takes from the browser. */
  nodeClass: string;
  icon: ReactElement;
  emptyContent: ReactNode;
}

/** A list of node IDs that the user fills by dragging nodes from the browser. */
const NodeList = ({
  path,
  title,
  nodeClass,
  icon,
  emptyContent,
}: NodeListProps): ReactElement => {
  const isPreview = Task.useIsPreview();
  const { set } = PForm.useContext();
  const nodes = PForm.useFieldValue<string[]>(path);
  const accepts = (item: Haul.Item): item is HaulItem =>
    isHaulItem(item) && item.data.nodeClass === nodeClass;
  const haulProps = Haul.useDrop({
    type: HAUL_TYPE,
    canDrop: ({ items }) => items.some(accepts),
    onDrop: ({ items }) => {
      const dropped = items.filter(accepts);
      set(path, [...new Set([...nodes, ...dropped.map(({ data }) => data.nodeId)])]);
      return dropped;
    },
  });
  const draggingState = Haul.useDraggingState();
  const isDragging = !isPreview && draggingState.items.some(accepts);
  return (
    <Flex.Box
      y
      empty
      className={CSS.cls(CSS.BE("opc-event", "nodes"), isDragging && CSS.M("dragging"))}
      {...(isPreview ? {} : haulProps)}
    >
      <Header.Header>
        <Header.Title weight={500} color={10} level="p">
          {title}
        </Header.Title>
      </Header.Header>
      <Flex.Box y gap="tiny" className={CSS.BE("opc-event", "content")}>
        {nodes.length === 0 ? (
          <Text.Text level="small" status="disabled">
            {emptyContent}
          </Text.Text>
        ) : (
          nodes.map((nodeId) => (
            <Flex.Box key={nodeId} x align="center" justify="between">
              <Text.Text level="small" color={10} gap="small" overflow="fade">
                {icon}
                {nodeId}
              </Text.Text>
              {!isPreview && (
                <Button.Button
                  variant="text"
                  size="small"
                  tooltip="Remove"
                  aria-label={`Remove ${nodeId}`}
                  onClick={() =>
                    set(
                      path,
                      nodes.filter((n) => n !== nodeId),
                    )
                  }
                >
                  <Icon.Close />
                </Button.Button>
              )}
            </Flex.Box>
          ))
        )}
      </Flex.Box>
    </Flex.Box>
  );
};

const FIELDS_PATH = "config.fields";

const BROWSE_PATH_INPUT_PROPS = { placeholder: "2:Level" } as const;

interface FieldItemProps {
  index: number;
}

const FieldItem = ({ index }: FieldItemProps): ReactElement | null => {
  const isPreview = Task.useIsPreview();
  const path = `${FIELDS_PATH}.${index}`;
  const field = PForm.useFieldValue<EventField>(path, { optional: true });
  const { remove } = PForm.useFieldListUtils<string, EventField>(FIELDS_PATH);
  if (field == null) return null;
  return (
    <Flex.Box x align="start" gap="small">
      <PForm.TextField path={`${path}.name`} label="Name" grow />
      <PForm.TextField
        path={`${path}.browsePath`}
        label="Browse path"
        grow
        inputProps={BROWSE_PATH_INPUT_PROPS}
      />
      {!isPreview && (
        <Button.Button
          variant="text"
          size="small"
          tooltip="Remove"
          aria-label={`Remove ${field.name === "" ? "field" : field.name}`}
          onClick={() => remove(field.key)}
          className={CSS.BE("opc-event", "remove-field")}
        >
          <Icon.Close />
        </Button.Button>
      )}
    </Flex.Box>
  );
};

const Fields = (): ReactElement => {
  const isPreview = Task.useIsPreview();
  const { data, push } = PForm.useFieldList<string, EventField>(FIELDS_PATH);
  const handleAdd = () => push({ key: id.create(), name: "", browsePath: "" });
  return (
    <Flex.Box y empty className={CSS.BE("opc-event", "fields")}>
      <Header.Header>
        <Header.Title weight={500} color={10} level="p">
          Fields
        </Header.Title>
        {!isPreview && (
          <Header.Actions>
            <Button.Button
              variant="text"
              size="medium"
              tooltip="Add field"
              aria-label="Add field"
              onClick={handleAdd}
            >
              <Icon.Add />
            </Button.Button>
          </Header.Actions>
        )}
      </Header.Header>
      <Flex.Box y gap="small" className={CSS.BE("opc-event", "content")}>
        <Text.Text level="small" color={9}>
          Each event records its ID, type, source, time, message, and severity. Add a
          field to record more of the event, by its browse path from the event type.
        </Text.Text>
        {data.map((key, i) => (
          <FieldItem key={key} index={i} />
        ))}
      </Flex.Box>
    </Flex.Box>
  );
};

// Event types are under the Types folder, beside the Objects folder that the browser
// starts at.
const BROWSER_ROOTS: ScannedNode[] = [
  {
    key: "i=3048",
    nodeId: "i=3048",
    name: "EventTypes",
    nodeClass: "Object",
    dataType: "",
    isArray: false,
    field: "",
  },
];

const Content = ({ device }: PlatformDevice.TaskFormContentProps<Device>) => {
  const isPreview = Task.useIsPreview();
  return (
    <>
      {!isPreview && <Browser device={device} roots={BROWSER_ROOTS} />}
      <Flex.Box y empty grow className={CSS.B("opc-event")}>
        <NodeList
          path="config.notifiers"
          title="Sources"
          nodeClass="Object"
          icon={<Icon.Group />}
          emptyContent="Records the events of the Server object. Drag objects from the browser to record their events instead."
        />
        <NodeList
          path="config.eventTypes"
          title="Event types"
          nodeClass="ObjectType"
          icon={<Icon.Type />}
          emptyContent="Records events of every type. Drag event types from the EventTypes folder of the browser to record only those types and their subtypes."
        />
        <Fields />
      </Flex.Box>
    </>
  );
};
Content.displayName = "OPCEventTaskForm";

const TaskForm: FC = PlatformDevice.wrapTaskForm({
  use,
  useConfigure: useConnectModal,
  Content,
});

const getInitialValues: Task.GetInitialValues<EventSchemas> = ({
  deviceKey,
  config,
}) => {
  const cfg = EVENT_SCHEMAS.config.parse(config ?? {});
  if (deviceKey != null) cfg.device = deviceKey;
  return { name: "OPC UA event task", type: EVENT_TYPE, config: cfg };
};

const channelExists = async (client: Synnax, key: channel.Key): Promise<boolean> => {
  if (key === 0) return false;
  try {
    await client.channels.retrieve(key);
    return true;
  } catch (e) {
    if (NotFoundError.matches(e)) return false;
    throw errors.fromUnknown(e);
  }
};

const onConfigure: Task.OnConfigure<EventSchemas["config"]> = async (
  client,
  config,
  name,
) => {
  const dev = await client.devices.retrieve({ key: config.device, schemas: SCHEMAS });
  if (await channelExists(client, config.channel)) return [config, dev.rack];
  const base = await findFreeName(
    client,
    `${channel.escapeInvalidName(name)}_events`,
    new Set(),
  );
  const index = await client.channels.create({
    name: `${base}_time`,
    dataType: DataType.TIMESTAMP,
    isIndex: true,
  });
  const events = await client.channels.create({
    name: base,
    dataType: DataType.JSON,
    index: index.key,
  });
  return [{ ...config, channel: events.key }, dev.rack];
};

export const Event = Task.wrapForm({
  Properties,
  Form: TaskForm,
  schemas: EVENT_SCHEMAS,
  deployConfigZ: deployEventConfigZ,
  type: EVENT_TYPE,
  getInitialValues,
  onConfigure,
});

export const useCreateEvent = Task.createUseCreate({ getInitialValues });

export const EventSelectable = Selector.createSelectable({
  type: EVENT_TYPE,
  title: "OPC UA event task",
  icon: <Icon.Logo.OPCUA />,
  useOnSelect: useCreateEvent,
});
