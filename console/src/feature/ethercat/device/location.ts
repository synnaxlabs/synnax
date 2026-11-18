// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { strings } from "@synnaxlabs/x";

import {
  type SlaveProperties,
  slavePropertiesZ,
} from "@/feature/ethercat/device/types";
import { Device } from "@/platform/device";

export const slotZ = slavePropertiesZ.pick({ networkName: true, position: true });

/** The fields of a slave that say where it is on the bus. */
export interface Slot extends Pick<SlaveProperties, "networkName" | "position"> {}

/**
 * @returns where the slave is, such as "Ethernet 2, slot 3". When the Driver names no
 * network, returns the location of the device.
 */
export const formatLocation = (
  location: string,
  { networkName, position }: Slot,
): string => (networkName == null ? location : `${networkName}, slot ${position}`);

const isIdentifier = (id: string): boolean => Device.identifierZ.safeParse(id).success;

/**
 * @returns identifiers that name the slot of the slave, such as "s3" and "slot3", then
 * the first that also names its network and fits, such as "e2_s3".
 */
export const suggestIdentifiers = ({ networkName, position }: Slot): string[] => {
  const slot = `s${position}`;
  const ids = [slot, `slot${position}`].filter(isIdentifier);
  const network = strings
    .createShortIdentifiers(networkName ?? "")
    .map((n) => `${n}_${slot}`)
    .find(isIdentifier);
  if (network != null) ids.push(network);
  return ids;
};
