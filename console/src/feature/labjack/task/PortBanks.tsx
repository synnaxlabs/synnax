// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/feature/labjack/task/PortBanks.css";

import { Flex } from "@synnaxlabs/lyra/flex";
import { Select } from "@synnaxlabs/lyra/select";
import { Text } from "@synnaxlabs/lyra/text";
import { type MouseEvent, type ReactElement, useState } from "react";

import { type Port } from "@/feature/labjack/device/types";
import {
  type Bank,
  banks,
  groups,
  type GroupsProps,
  numbers,
  terminal,
} from "@/feature/labjack/task/ports";
import { CSS } from "@/platform/css";
import { Task } from "@/platform/task";

interface GridProps extends GroupsProps {
  onClick?: (cell: Task.Port.Cell) => void;
  /** Moves a channel to the port: the grid is a picker. */
  onMove?: (cell: Task.Port.Cell) => void;
  /** Adds a channel on a free port. */
  onAdd?: (cell: Task.Port.Cell) => void;
  /** Opens a menu over a free port. */
  onContextMenu?: (e: MouseEvent) => void;
}

const Grid = ({
  bank,
  onClick,
  onMove,
  onAdd,
  onContextMenu,
  ...rest
}: GridProps): ReactElement => (
  <Task.Port.Grid
    label={`${terminal(bank.ports[0])} to ${terminal(bank.ports[bank.ports.length - 1])}`}
    groups={groups({ bank, ...rest })}
    onClick={onClick}
    onMove={onMove}
    onAdd={onAdd}
    onContextMenu={onContextMenu}
  />
);

export interface PortBanksProps extends Omit<GridProps, "bank"> {
  ports: Port[];
  /** Each bank's title beside its grid, for a map above a list. */
  compact?: boolean;
}

/** The ports as a grid for each bank, each under or, when compact, beside its title. */
export const PortBanks = ({
  ports,
  compact = false,
  ...rest
}: PortBanksProps): ReactElement => {
  if (compact)
    return (
      <Task.Port.Rows>
        {banks(ports).map((bank) => (
          <Task.Port.Row key={bank.ports[0].key} label={bank.title}>
            <Grid bank={bank} {...rest} />
          </Task.Port.Row>
        ))}
      </Task.Port.Rows>
    );
  return (
    <div className={CSS.B("labjack-port-banks")}>
      {banks(ports).map((bank) => (
        <Flex.Box key={bank.ports[0].key} y gap="tiny">
          <Text.Text level="small" color={9}>
            {bank.title}
          </Text.Text>
          <Grid bank={bank} {...rest} />
        </Flex.Box>
      ))}
    </div>
  );
};

export interface BankTabsProps extends Omit<PortBanksProps, "compact"> {
  /** The key of the channel's port. The tabs open on its bank. */
  port: string;
}

/**
 * The ports one bank at a time, under a tab for each bank that names it and its port
 * numbers, such as "X3 48–71".
 */
export const BankTabs = ({ ports, port, ...rest }: BankTabsProps): ReactElement => {
  const [viewed, setViewed] = useState<string>();
  const list = banks(ports);
  const bank: Bank =
    list.find(({ name }) => name === viewed) ??
    list.find((b) => b.ports.some(({ key }) => key === port)) ??
    list[0];
  return (
    <Flex.Box y gap="small">
      <Select.Buttons<string> size="small" wrap value={bank.name} onChange={setViewed}>
        {list.map((b) => (
          <Select.Item<string> key={b.name} itemKey={b.name} size="small">
            {b.name}{" "}
            <Text.Text level="small" color={9}>
              {numbers(b)}
            </Text.Text>
          </Select.Item>
        ))}
      </Select.Buttons>
      <Grid bank={bank} {...rest} />
    </Flex.Box>
  );
};
