// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type device, type library, mil1553 } from "@synnaxlabs/client";
import { Icon } from "@synnaxlabs/lyra/icon";

import { Bus } from "@/feature/bus";
import { Select } from "@/feature/mil1553/device/device";

const checkIdentifier = Bus.checkIdentifier("mil1553", "MIL-STD-1553");

/**
 * @returns the command and device properties of a message that passed checkIdentifier.
 * @throws {Error} if the message has no MIL-STD-1553 identifier.
 */
const parse = (
  { name, identifier }: library.MessageEntry,
  dev: device.Device,
): [library.Mil1553Identifier, mil1553.Properties] => {
  if (identifier?.type !== "mil1553")
    throw new Error(`Message ${name} has no MIL-STD-1553 identifier`);
  return [identifier, mil1553.propertiesZ.parse(dev.properties)];
};

const checkOwned = (
  name: string,
  { rt }: library.Mil1553Identifier,
  { terminals }: mil1553.Properties,
): string | null =>
  terminals.includes(rt)
    ? null
    : `Message ${name} is for terminal ${rt}, which the remote terminal does not own`;

/** @param use - What the role does with the message, such as "read". */
const checkDirection = (
  name: string,
  id: library.Mil1553Identifier,
  direction: library.Direction,
  role: string,
  use: string,
): string | null =>
  id.direction === direction
    ? null
    : `Message ${name} must be a ${direction} message for a ${role} to ${use}`;

const checkRead: Bus.MessageCheck = (entry, dev) => {
  const [id, props] = parse(entry, dev);
  const { name, period } = entry;
  switch (props.role) {
    case "bus_controller":
      return (
        checkDirection(name, id, "transmit", "bus controller", "read") ??
        (period == null
          ? `Message ${name} needs a period for the bus controller to poll it`
          : null)
      );
    case "remote_terminal":
      return (
        checkDirection(name, id, "receive", "remote terminal", "read") ??
        checkOwned(name, id, props)
      );
    case "monitor":
      return null;
  }
};

const checkWrite: Bus.MessageCheck = (entry, dev) => {
  const [id, props] = parse(entry, dev);
  const { name } = entry;
  switch (props.role) {
    case "bus_controller":
      return checkDirection(name, id, "receive", "bus controller", "send");
    case "remote_terminal":
      return (
        checkDirection(name, id, "transmit", "remote terminal", "answer with") ??
        checkOwned(name, id, props)
      );
    case "monitor":
      return `Message ${name} cannot be sent by a monitor`;
  }
};

export const {
  PREFIX,
  READ_TYPE,
  WRITE_TYPE,
  READ_SCHEMAS,
  WRITE_SCHEMAS,
  Read,
  Write,
  useCreateRead,
  useCreateWrite,
  COMMANDS,
  SELECTABLES,
  FORMS,
} = Bus.createTasks({
  prefix: "mil1553",
  name: "MIL-STD-1553",
  icon: <Icon.Node />,
  readConfigZ: mil1553.readConfigZ,
  writeConfigZ: mil1553.writeConfigZ,
  accepts: (entry) => checkIdentifier(entry) == null,
  SelectDevice: Select,
  readMessageChecks: [checkIdentifier, checkRead],
  writeMessageChecks: [checkIdentifier, checkWrite],
});
