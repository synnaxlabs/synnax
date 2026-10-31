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
  type Claims,
  describer,
  digitalPorts,
  groups,
  holdOf,
  lineClaims,
  lineKey,
  type PortKind,
  usedCount,
} from "@/feature/ni/task/ports";
import { type DigitalChannel } from "@/feature/ni/task/types";
import { ContextMenu } from "@/platform/context-menu";
import { CSS } from "@/platform/css";
import { Task } from "@/platform/task";

const CHANNELS_PATH = "config.channels";

const DEVICE_MENU_KEY = "device";

interface DeviceSectionProps {
  device: Device;
  /** What the device's ports hold, such as "3 of 16 ports used". */
  usage: string;
  editable: boolean;
  /** Adds a channel on the free cell with the key, or on the lowest free port. */
  onAdd: (cell?: string) => void;
  onRemove?: () => void;
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
      <Flex.Box y gap="medium">
        <Flex.Box
          y
          gap="tiny"
          className={Menu.CONTEXT_TARGET}
          data-menu-key={DEVICE_MENU_KEY}
          onContextMenu={open}
        >
          <Text.Text level="p" weight={500} color={11}>
            {device.model}
          </Text.Text>
          <Text.Text level="small" color={9} className={CSS.BE("ni-ports", "usage")}>
            {`${device.location} · ${usage}`}
          </Text.Text>
        </Flex.Box>
        {children(open)}
      </Flex.Box>
    </Menu.ContextMenu>
  );
};

interface DevicePortsProps<C extends Task.Channel> extends Task.Views.PortsProps {
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
  const claims: Claims =
    hardware == null ? new Map() : kind.claims(channels, device.key, hardware);
  const handleClick = (cell: Task.PortCell) => {
    if (cell.channel != null) onSelect([cell.channel]);
    else if (editable) onAdd(Number(cell.key));
  };
  return (
    <DeviceSection
      device={device}
      usage={
        hardware == null
          ? "No port list yet"
          : `${usedCount(ports, claims)} of ${ports.length} ports used`
      }
      editable={editable}
      onAdd={(cell) => onAdd(cell == null ? undefined : Number(cell))}
      onRemove={onRemove}
    >
      {(onContextMenu) =>
        hardware != null && (
          <Task.PortGrid
            label={`${device.name} ports`}
            groups={groups({
              ports,
              claims,
              prefix: kind.prefix,
              selected,
              describe: describer(kind, channels, device),
            })}
            onClick={handleClick}
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
    initialQuery={{ makes: [MAKE], racks: [rack] }}
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

export interface PortsViewProps<C extends Task.Channel> extends Task.Views.PortsProps {
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
  kind,
  devices,
  deviceOf,
  create,
  rack,
  selected,
  onSelect,
}: PortsViewProps<C>): ReactElement => {
  const { get } = Form.useContext();
  const { push, remove } = Form.useFieldList<string, C>(CHANNELS_PATH);
  const channels = Form.useFieldValue<C[]>(CHANNELS_PATH);
  const editable = !Task.useIsPreview();
  const add = (device: device.Key, port?: number) => {
    const channel = create(get<C[]>(CHANNELS_PATH).value, device, port);
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

export interface LinesViewProps<C extends DigitalChannel>
  extends Task.Views.PortsProps {
  device: Device;
  direction: "input" | "output";
  /** Creates a channel on the line, or on the lowest free line. */
  create: (channels: C[], placement: LinePlacement) => C;
  /** The name a channel shows by. */
  name: (channel: C, device: Device) => string;
}

/**
 * A digital task's channels by line: one row of lines for each port of the device. A
 * click on a used line selects its channel, and a click on a free line adds a channel
 * on it.
 */
export const LinesView = <C extends DigitalChannel>({
  device,
  direction,
  create,
  name,
  selected,
  onSelect,
}: LinesViewProps<C>): ReactElement => {
  const { get } = Form.useContext();
  const { push } = Form.useFieldList<string, C>(CHANNELS_PATH);
  const channels = Form.useFieldValue<C[]>(CHANNELS_PATH);
  const editable = !Task.useIsPreview();
  const { hardware } = device.properties;
  const ports = hardware == null ? [] : digitalPorts(hardware, direction);
  const claims = lineClaims(channels);
  const add = (placement: LinePlacement = {}) => {
    const channel = create(get<C[]>(CHANNELS_PATH).value, placement);
    push(channel);
    onSelect([channel.key]);
  };
  const parse = (cell: string): LinePlacement => {
    const [port, line] = cell.split("/").map(Number);
    return { port, line };
  };
  const describe = (key: string) => {
    const channel = channels.find((c) => c.key === key);
    return channel == null ? "" : name(channel, device);
  };
  const cell = (port: number, line: number): Task.PortCell => {
    const key = lineKey(port, line);
    const list = claims.get(key) ?? [];
    const { state, disabled, channel, names } = holdOf([list], describe);
    const lineName = `port${port}/line${line}`;
    let tooltip = `${lineName} is free`;
    if (list.length === 1) tooltip = `${lineName}: ${names}`;
    else if (list.length > 1) tooltip = `${lineName} is used by ${names}`;
    return {
      key,
      name: lineName,
      labels: [String(line)],
      row: 1,
      column: line + 1,
      state,
      disabled,
      selected: list.some((c) => selected.includes(c.channel)),
      channel,
      tooltip,
    };
  };
  const handleClick = ({ key, channel }: Task.PortCell) => {
    if (channel != null) onSelect([channel]);
    else if (editable) add(parse(key));
  };
  const lines = new Set(
    ports.flatMap(({ port, lines: n }) =>
      Array.from({ length: n }, (_, line) => lineKey(port, line)),
    ),
  );
  const used = [...claims.keys()].filter((k) => lines.has(k)).length;
  return (
    <Flex.Box y empty className={CSS.B("ni-ports")}>
      <DeviceSection
        device={device}
        usage={
          hardware == null ? "No port list yet" : `${used} of ${lines.size} lines used`
        }
        editable={editable}
        onAdd={(c) => add(c == null ? undefined : parse(c))}
      >
        {(onContextMenu) => (
          <Flex.Box y gap="small">
            {ports.map(({ port, lines: n }) => (
              <Flex.Box key={port} x align="center" gap="medium">
                <Text.Text
                  level="small"
                  color={9}
                  className={CSS.BE("ni-ports", "port")}
                >
                  {`port${port}`}
                </Text.Text>
                <Task.PortGrid
                  label={`port${port} lines`}
                  groups={[
                    {
                      key: "lines",
                      columns: n,
                      cells: Array.from({ length: n }, (_, line) => cell(port, line)),
                    },
                  ]}
                  onClick={handleClick}
                  onContextMenu={onContextMenu}
                />
              </Flex.Box>
            ))}
          </Flex.Box>
        )}
      </DeviceSection>
    </Flex.Box>
  );
};
