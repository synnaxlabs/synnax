// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Flex } from "@synnaxlabs/lyra/flex";
import { Text } from "@synnaxlabs/lyra/text";
import { type MouseEvent, type ReactElement } from "react";

import { type Port } from "@/feature/labjack/device/types";
import {
  banks,
  groups,
  type GroupsProps,
  terminal,
} from "@/feature/labjack/task/ports";
import { Task } from "@/platform/task";

export interface PortBanksProps extends Omit<GroupsProps, "bank"> {
  ports: Port[];
  onClick: (cell: Task.PortCell) => void;
  /** Opens a menu over a free port. */
  onContextMenu?: (e: MouseEvent) => void;
}

/** The ports as a grid for each bank, each under its title. */
export const PortBanks = ({
  ports,
  onClick,
  onContextMenu,
  ...rest
}: PortBanksProps): ReactElement => (
  <Flex.Box y gap="medium">
    {banks(ports).map((bank) => (
      <Flex.Box key={bank.ports[0].key} y gap="tiny">
        <Text.Text level="small" color={9}>
          {bank.title}
        </Text.Text>
        <Task.PortGrid
          label={`${terminal(bank.ports[0])} to ${terminal(bank.ports[bank.ports.length - 1])}`}
          groups={groups({ bank, ...rest })}
          onClick={onClick}
          onContextMenu={onContextMenu}
        />
      </Flex.Box>
    ))}
  </Flex.Box>
);
