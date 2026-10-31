// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/feature/labjack/task/PortsView.css";

import { Flex } from "@synnaxlabs/lyra/flex";
import { Form } from "@synnaxlabs/lyra/form";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Menu } from "@synnaxlabs/lyra/menu";
import { Text } from "@synnaxlabs/lyra/text";
import { type ReactElement } from "react";

import {
  AI_PORT_TYPE,
  AO_PORT_TYPE,
  DI_PORT_TYPE,
  DO_PORT_TYPE,
  type Model,
  type Port,
  type PortType,
} from "@/feature/labjack/device/types";
import { PortBanks } from "@/feature/labjack/task/PortBanks";
import {
  type Claims,
  claims,
  describer,
  terminal,
  usedCount,
} from "@/feature/labjack/task/ports";
import { type Channel } from "@/feature/labjack/task/types";
import { ContextMenu } from "@/platform/context-menu";
import { CSS } from "@/platform/css";
import { Task } from "@/platform/task";

const CHANNELS_PATH = "config.channels";

const TITLES: Record<PortType, string> = {
  [AI_PORT_TYPE]: "Analog inputs",
  [AO_PORT_TYPE]: "Analog outputs",
  [DI_PORT_TYPE]: "Digital inputs",
  [DO_PORT_TYPE]: "Digital outputs",
};

interface SectionProps<P extends Port> {
  type: PortType;
  ports: P[];
  claims: Claims;
  selected: string[];
  describe: (channel: string) => string;
  editable: boolean;
  onSelect: (keys: string[]) => void;
  onAdd: (port: P) => void;
}

/**
 * The ports of one type and how many hold channels, over their banks. A right click on
 * a free port offers "Add channel", and one on the title adds on the lowest free port.
 */
const Section = <P extends Port>({
  type,
  ports,
  claims,
  selected,
  describe,
  editable,
  onSelect,
  onAdd,
}: SectionProps<P>): ReactElement => {
  const menu = Menu.useContextMenu();
  const portOf = (key: string): P => {
    const port = ports.find((p) => p.key === key);
    if (port == null) throw new Error(`LabJack port ${key} is not a ${type} port`);
    return port;
  };
  const free = ports.find((p) => !claims.has(terminal(p)));
  const renderMenu = ({ keys: [key] }: Menu.ContextMenuMenuProps) => (
    <ContextMenu.Menu>
      <Menu.Item itemKey="add" onClick={() => onAdd(portOf(key))}>
        <Icon.Add />
        Add channel
      </Menu.Item>
    </ContextMenu.Menu>
  );
  const open = editable ? menu.open : undefined;
  const handleClick = (cell: Task.PortCell) => {
    if (cell.channel != null) onSelect([cell.channel]);
    else if (editable) onAdd(portOf(cell.key));
  };
  return (
    <Menu.ContextMenu {...menu} menu={renderMenu}>
      <Flex.Box y gap="medium" role="group" aria-label={TITLES[type]}>
        <Flex.Box
          y
          gap="tiny"
          className={free == null ? undefined : Menu.CONTEXT_TARGET}
          data-menu-key={free?.key}
          onContextMenu={free == null ? undefined : open}
        >
          <Text.Text level="p" weight={500} color={11}>
            {TITLES[type]}
          </Text.Text>
          <Text.Text
            level="small"
            color={9}
            className={CSS.BE("labjack-ports", "usage")}
          >
            {`${usedCount(ports, claims)} of ${ports.length} ports used`}
          </Text.Text>
        </Flex.Box>
        <PortBanks
          ports={ports}
          claims={claims}
          selected={selected}
          describe={describe}
          onClick={handleClick}
          onContextMenu={open}
        />
      </Flex.Box>
    </Menu.ContextMenu>
  );
};

export interface PortsViewProps<C extends Channel, P extends Port>
  extends Task.Views.PortsProps {
  model: Model;
  /** The ports the task's channels can use. Each type shows in a section of its own. */
  ports: P[];
  create: (channels: C[], port: P) => C;
  /** The name a channel shows by. */
  name: (channel: C) => string;
}

/**
 * The task's channels by port: for each port type, how many ports hold channels and the
 * ports by bank. A click on a used port selects its channel, and a click on a free port
 * adds a channel on it.
 */
export const PortsView = <C extends Channel, P extends Port>({
  model,
  ports,
  create,
  name,
  selected,
  onSelect,
}: PortsViewProps<C, P>): ReactElement => {
  const { get } = Form.useContext();
  const { push } = Form.useFieldList<string, C>(CHANNELS_PATH);
  const channels = Form.useFieldValue<C[]>(CHANNELS_PATH);
  const editable = !Task.useIsPreview();
  const portClaims = claims(channels, model);
  const types = [...new Set(ports.map(({ type }) => type))];
  const add = (port: P) => {
    const channel = create(get<C[]>(CHANNELS_PATH).value, port);
    push(channel);
    onSelect([channel.key]);
  };
  return (
    <Flex.Box y empty className={CSS.B("labjack-ports")}>
      {types.map((type) => (
        <Section<P>
          key={type}
          type={type}
          ports={ports.filter((p) => p.type === type)}
          claims={portClaims}
          selected={selected}
          describe={describer(channels, name)}
          editable={editable}
          onSelect={onSelect}
          onAdd={add}
        />
      ))}
    </Flex.Box>
  );
};
