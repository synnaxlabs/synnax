// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/feature/ni/task/PortsView.css";

import { type device, type rack } from "@synnaxlabs/client";
import { Flex } from "@synnaxlabs/lyra/flex";
import { Form } from "@synnaxlabs/lyra/form";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Menu } from "@synnaxlabs/lyra/menu";
import { Text } from "@synnaxlabs/lyra/text";
import { Device as PDevice } from "@synnaxlabs/pluto";
import { type MouseEvent, type ReactElement, type ReactNode } from "react";

import { type Device, MAKE } from "@/feature/ni/device/types";
import { type LinePlacement } from "@/feature/ni/task/createChannel";
import {
  claimsOf,
  digitalPorts,
  groups,
  lineClaims,
  lineName,
  type PortKind,
} from "@/feature/ni/task/ports";
import { type DigitalChannel } from "@/feature/ni/task/types";
import { ContextMenu } from "@/platform/context-menu";
import { CSS } from "@/platform/css";
import { Task } from "@/platform/task";

const DEVICE_MENU_KEY = "device";

const lineKey = (port: number, line: number): string => `${port}/${line}`;

const LINES_PER_ROW = 8;

interface DeviceSectionProps {
  device: Device;
  /** What the device's ports hold, such as "3 of 16 ports used". */
  usage: string;
  editable: boolean;
  /** Adds a channel on the free cell with the key, or on the lowest free port. */
  onAdd: (cell?: string) => void;
  onRemove?: () => void;
  /** The title on one line, for a map above a list. */
  compact?: boolean;
  /** The device's grid, given the handler that opens the menu of a free cell. */
  children: (onContextMenu?: (e: MouseEvent) => void) => ReactNode;
}

/**
 * A device's model, location, and usage over its port grid. A right click on a free
 * port offers "Add channel", and one on the device also offers "Remove device".
 */
const DeviceSection = ({
  device,
  usage,
  editable,
  onAdd,
  onRemove,
  compact = false,
  children,
}: DeviceSectionProps): ReactElement => {
  const menu = Menu.useContextMenu();
  const renderMenu = ({ keys: [key] }: Menu.ContextMenuMenuProps) => (
    <ContextMenu.Menu>
      <Menu.Item
        itemKey="add"
        onClick={() => onAdd(key === DEVICE_MENU_KEY ? undefined : key)}
      >
        <Icon.Add />
        Add channel
      </Menu.Item>
      {key === DEVICE_MENU_KEY && onRemove != null && (
        <>
          <Menu.Divider />
          <Menu.Item itemKey="remove" onClick={onRemove}>
            <Icon.Close />
            Remove device
          </Menu.Item>
        </>
      )}
    </ContextMenu.Menu>
  );
  const open = editable ? menu.open : undefined;
  return (
    <Menu.ContextMenu {...menu} menu={renderMenu}>
      <Flex.Box y gap={compact ? "small" : "medium"}>
        <Flex.Box
          x
          justify="between"
          align={compact ? "center" : "start"}
          gap="small"
          className={Menu.CONTEXT_TARGET}
          data-menu-key={DEVICE_MENU_KEY}
          onContextMenu={open}
        >
          <Flex.Box
            direction={compact ? "x" : "y"}
            wrap={compact}
            align={compact ? "center" : undefined}
            gap={compact ? "small" : "tiny"}
          >
            <Text.Text
              level={compact ? "small" : "p"}
              weight={500}
              color={compact ? 10 : 11}
            >
              {device.model}
            </Text.Text>
            <Text.Text level="small" color={9} className={CSS.BE("ni-ports", "usage")}>
              {usage}
            </Text.Text>
          </Flex.Box>
          <span className={CSS.BE("ni-ports", "location")}>{device.location}</span>
        </Flex.Box>
        {children(open)}
      </Flex.Box>
    </Menu.ContextMenu>
  );
};

interface DevicePortsProps<C extends Task.Channel> extends Omit<
  Task.Views.PortsProps<C>,
  "path"
> {
  device: Device;
  kind: PortKind<C>;
  channels: C[];
  editable: boolean;
  onAdd: (port?: number) => void;
  onRemove?: () => void;
}

const DevicePorts = <C extends Task.Channel>({
  device,
  kind,
  channels,
  editable,
  selected,
  onSelect,
  onAdd,
  onRemove,
}: DevicePortsProps<C>): ReactElement => {
  const { hardware } = device.properties;
  const ports = hardware == null ? [] : kind.ports(hardware);
  const claims = claimsOf(kind, channels, hardware);
  const names = ports.map(({ port }) => `${kind.prefix}${port}`);
  const handleClick = ({ channel }: Task.Port.Cell) => {
    if (channel != null) onSelect([channel]);
  };
  return (
    <DeviceSection
      device={device}
      usage={
        hardware == null
          ? "No port list yet"
          : `${Task.Port.usedCount(names, claims)} of ${ports.length} ports used`
      }
      editable={editable}
      onAdd={(cell) => onAdd(cell == null ? undefined : Number(cell))}
      onRemove={onRemove}
    >
      {(onContextMenu) =>
        hardware != null && (
          <Task.Port.Grid
            label={`${device.name} ports`}
            groups={groups({
              ports,
              claims,
              prefix: kind.prefix,
              selected,
              describe: Task.Port.describer(channels, (c) => kind.name(c, device)),
            })}
            onClick={handleClick}
            onAdd={editable ? ({ key }) => onAdd(Number(key)) : undefined}
            onContextMenu={onContextMenu}
          />
        )
      }
    </DeviceSection>
  );
};

interface AddDeviceProps {
  rack: rack.Key;
  exclude: device.Key[];
  onAdd: (key: device.Key) => void;
}

const AddDevice = ({ rack, exclude, onAdd }: AddDeviceProps): ReactElement => (
  <PDevice.SelectSingle
    value={undefined}
    onChange={(key: device.Key) => onAdd(key)}
    query={{ where: { make: { eq: MAKE }, rack: { eq: rack } } }}
    filter={(d) =>
      d.make === MAKE && d.rack === rack && d.configured && !exclude.includes(d.key)
    }
    emptyContent="No other NI devices on this rack"
    className={CSS.BE("ni-ports", "add")}
    triggerProps={{
      placeholder: "Add device",
      variant: "text",
      icon: <Icon.Add />,
      hideCaret: true,
      className: CSS.cls(
        CSS.B("create-list-item"),
        CSS.M("small"),
        CSS.BE("ni-ports", "trigger"),
      ),
    }}
  />
);

export interface PortsViewProps<C extends Task.Channel> extends Task.Views
  .PortsProps<C> {
  kind: PortKind<C>;
  /** The devices the task uses. They show in the order of their first channels. */
  devices: Device[];
  deviceOf: (channel: C) => device.Key;
  /** Creates a channel on the device, on the port or the lowest free one. */
  create: (channels: C[], device: device.Key, port?: number) => C;
  /**
   * The rack of the task's devices. With it, the view adds and removes devices; a task
   * that names its one device in its properties leaves it out.
   */
  rack?: rack.Key;
}

/**
 * The task's channels by port: each device's model, its location and how many ports
 * hold channels, and its port grid. A click on a used port selects its channel, and a
 * click on a free port adds a channel on it.
 */
export const PortsView = <C extends Task.Channel>({
  path,
  kind,
  devices,
  deviceOf,
  create,
  rack,
  selected,
  onSelect,
}: PortsViewProps<C>): ReactElement => {
  const { get } = Form.useContext();
  const { push, remove } = Form.useFieldList(path);
  const channels = Form.useFieldValue(path);
  const editable = !Task.useIsPreview();
  const add = (device: device.Key, port?: number) => {
    const channel = create(get(path).value, device, port);
    push(channel);
    onSelect([channel.key]);
  };
  const removeDevice = (device: device.Key) => {
    remove(channels.filter((c) => deviceOf(c) === device).map(({ key }) => key));
    onSelect([]);
  };
  const first = (key: device.Key) => channels.findIndex((c) => deviceOf(c) === key);
  const ordered = [...devices].sort((a, b) => first(a.key) - first(b.key));
  return (
    <>
      <Flex.Box y empty className={CSS.B("ni-ports")}>
        {ordered.map((device) => (
          <DevicePorts<C>
            key={device.key}
            device={device}
            kind={kind}
            channels={channels.filter((c) => deviceOf(c) === device.key)}
            editable={editable}
            selected={selected}
            onSelect={onSelect}
            onAdd={(port) => add(device.key, port)}
            onRemove={rack == null ? undefined : () => removeDevice(device.key)}
          />
        ))}
      </Flex.Box>
      {rack != null && editable && (
        <AddDevice
          rack={rack}
          exclude={devices.map(({ key }) => key)}
          onAdd={(key) => add(key)}
        />
      )}
    </>
  );
};

export interface LinesViewProps<C extends DigitalChannel> extends Task.Views
  .PortsProps<C> {
  device: Device;
  direction: "input" | "output";
  /** Creates a channel on the line, or on the lowest free line. */
  create: (channels: C[], placement: LinePlacement) => C;
  /** The name a channel shows by. */
  name: (channel: C, device: Device) => string;
}

/**
 * A digital task's channels by line, as a map above its list: the device's ports in a
 * line, eight lines to a row. A click on a used line selects its channel, and a click
 * on a free line adds a channel on it.
 */
export const LinesView = <C extends DigitalChannel>({
  path,
  device,
  direction,
  create,
  name,
  selected,
  onSelect,
}: LinesViewProps<C>): ReactElement => {
  const { get } = Form.useContext();
  const { push } = Form.useFieldList(path);
  const channels = Form.useFieldValue(path);
  const editable = !Task.useIsPreview();
  const { hardware } = device.properties;
  const ports = hardware == null ? [] : digitalPorts(hardware, direction);
  const claims = lineClaims(channels);
  const add = (placement: LinePlacement = {}) => {
    const channel = create(get(path).value, placement);
    push(channel);
    onSelect([channel.key]);
  };
  const parse = (cell: string): LinePlacement => {
    const [port, line] = cell.split("/").map(Number);
    return { port, line };
  };
  const describe = Task.Port.describer(channels, (c) => name(c, device));
  const cell = (port: number, line: number): Task.Port.Cell =>
    Task.Port.cell({
      key: lineKey(port, line),
      labels: [String(line)],
      row: Math.floor(line / LINES_PER_ROW) + 1,
      column: (line % LINES_PER_ROW) + 1,
      ports: [lineName(port, line)],
      claims,
      selected,
      describe,
    });
  const handleClick = ({ channel }: Task.Port.Cell) => {
    if (channel != null) onSelect([channel]);
  };
  const lines = ports.flatMap(({ port, lines: n }) =>
    Array.from({ length: n }, (_, line) => lineName(port, line)),
  );
  const used = Task.Port.usedCount(lines, claims);
  return (
    <Flex.Box
      y
      empty
      className={CSS.cls(CSS.B("ni-ports"), CSS.BM("ni-ports", "compact"))}
    >
      <DeviceSection
        device={device}
        compact
        usage={
          hardware == null
            ? "No port list yet"
            : `${used} of ${lines.length} lines used`
        }
        editable={editable}
        onAdd={(c) => add(c == null ? undefined : parse(c))}
      >
        {(onContextMenu) => (
          <Task.Port.Rows>
            {ports.map(({ port, lines: n }) => (
              <Task.Port.Row key={port} label={`port${port}`}>
                <Task.Port.Grid
                  label={`port${port} lines`}
                  groups={[
                    {
                      key: String(port),
                      columns: Math.min(LINES_PER_ROW, n),
                      cells: Array.from({ length: n }, (_, line) => cell(port, line)),
                    },
                  ]}
                  onClick={handleClick}
                  onAdd={editable ? ({ key }) => add(parse(key)) : undefined}
                  onContextMenu={onContextMenu}
                />
              </Task.Port.Row>
            ))}
          </Task.Port.Rows>
        )}
      </DeviceSection>
    </Flex.Box>
  );
};
