// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ethercat } from "@synnaxlabs/client";
import { Form } from "@synnaxlabs/lyra/form";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Menu } from "@synnaxlabs/lyra/menu";
import { primitive } from "@synnaxlabs/x";
import { type ReactElement } from "react";

import { useResultSlave } from "@/feature/ethercat/device/queries";
import {
  addressKey,
  type Channel,
  DIRECTIONS,
  hasEntry,
  type PDOType,
  uniqueEntries,
} from "@/feature/ethercat/task/types";
import { type Task } from "@/platform/task";

const CHANNELS_PATH = "config.channels";

export interface AddSlaveEntriesItemProps<
  C extends Channel,
> extends Task.ContextMenuItemProps<C> {
  /** The entries to add: inputs for a read task, outputs for a write task. */
  pdoType: PDOType;
  /** Creates a channel bound to one entry of the slave with the given key. */
  createChannel: (device: string, entry: ethercat.PDOEntry) => C;
}

/**
 * A context menu item that adds a channel for every entry of the selected channel's
 * slave that no channel of the task uses yet. Renders nothing unless exactly one
 * channel with a slave is selected and the slave has an unused entry.
 */
export const AddSlaveEntriesItem = <C extends Channel>({
  channels,
  keys,
  pdoType,
  createChannel,
}: AddSlaveEntriesItemProps<C>): ReactElement | null => {
  const device = keys.length === 1 ? (channels[0]?.device ?? "") : "";
  const { data: slave } = useResultSlave(
    primitive.isZero(device) ? null : { key: device },
  );
  const all = Form.useFieldValue<C[]>(CHANNELS_PATH);
  const { push } = Form.useFieldListUtils<string, C>(CHANNELS_PATH);
  if (slave == null) return null;
  const used = new Set(
    all.flatMap((ch) => (ch.device === device && hasEntry(ch) ? [addressKey(ch)] : [])),
  );
  const unused = uniqueEntries(slave.properties.pdos[pdoType]).filter(
    (entry) => !used.has(addressKey(entry)),
  );
  if (unused.length === 0) return null;
  const handleClick = () => push(unused.map((entry) => createChannel(device, entry)));
  return (
    <>
      <Menu.Item itemKey="addSlaveEntries" onClick={handleClick}>
        <Icon.Add />
        {`Add every ${DIRECTIONS[pdoType]} of ${slave.name}`}
      </Menu.Item>
      <Menu.Divider />
    </>
  );
};
