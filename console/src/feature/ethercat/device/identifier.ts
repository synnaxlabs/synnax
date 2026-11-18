// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { caseconv } from "@synnaxlabs/x";

import { MAKE, SLAVE_MODEL } from "@/feature/ethercat/device/types";
import { Device } from "@/platform/device";

/** The fields of a device of any make that the identifier checks read. */
export interface Holder {
  key: string;
  name: string;
  make: string;
  model: string;
  properties?: { identifier?: unknown; position?: unknown };
}

/**
 * @returns the device other than the slave whose identifier gives the same channel
 * names, or undefined. A slave with no identifier clashes with no device.
 */
export const findClash = <D extends Holder>(
  slave: Holder,
  devices: D[],
): D | undefined => {
  const prefix = Device.channelPrefix(slave);
  if (prefix == null) return undefined;
  return devices.find((d) => d.key !== slave.key && Device.channelPrefix(d) === prefix);
};

/** @returns "module <name> in slot <position>" for a slave, else "device <name>". */
const nameOf = (d: Holder): string => {
  const position = d.properties?.position;
  if (d.make === MAKE && d.model === SLAVE_MODEL && typeof position === "number")
    return `module ${d.name} in slot ${position}`;
  return `device ${d.name}`;
};

/** @returns why the slave cannot start while the other device has its identifier. */
export const clashProblem = (slave: Holder, other: Holder): string =>
  `${caseconv.capitalize(nameOf(other))} has an identifier that gives the same ` +
  `channel names as ${nameOf(slave)}. In Configure, change the identifier of one of ` +
  "them.";

/** @returns why a slave cannot take an identifier that the other device has. */
export const clashMessage = (other: Holder): string =>
  `${caseconv.capitalize(nameOf(other))} has an identifier that gives the same ` +
  "channel names. Pick another identifier.";
