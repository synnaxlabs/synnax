// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Address, sameAddress } from "@/feature/ethercat/device/settings";
import { type SlaveProperties } from "@/feature/ethercat/device/types";

/**
 * How the scan found a slave set up:
 * - synnax: Synnax alone sets it up, or its catalog describes no mode.
 * - other: it sends data that only reader reads.
 * - shared: reader sets it up, and Synnax uses its data.
 * - bus: it runs Synnax only, which uses bus 1 IN, but Synnax is on bus 2 IN.
 * - switched: it holds Synnax only, but it sends data that only reader reads until its
 *   power restarts.
 */
export type Setup =
  | { kind: "synnax" }
  | { kind: "bus" }
  | { kind: "other"; reader: string }
  | { kind: "shared"; reader: string }
  | { kind: "switched"; reader: string };

interface Marked {
  address: Address;
  reader: string;
}

/** @returns each entry of the catalog that only one program reads. */
const markedEntries = (props: SlaveProperties): Marked[] =>
  (props.settings?.channels ?? []).flatMap(({ entries }) =>
    entries.flatMap((address) =>
      address.onlyReadBy == null ? [] : [{ address, reader: address.onlyReadBy }],
    ),
  );

/**
 * @returns the one program that reads the input entry at the address, or null when
 * Synnax can read it.
 */
export const onlyReaderOf = (props: SlaveProperties, address: Address): string | null =>
  markedEntries(props).find((m) => sameAddress(m.address, address))?.reader ??
  props.settings?.mode?.onlyReadBy ??
  null;

/** @returns whether other software sets up the slave, so Synnax writes no settings. */
export const isShared = (props: SlaveProperties): boolean =>
  props.settings?.mode?.shared === true;

/** @returns the software that sets up the slave, or null when Synnax does. */
export const setUpBy = (props: SlaveProperties): string | null => {
  const mode = props.settings?.mode;
  if (mode?.shared !== true) return null;
  // A Driver before the bus and the software were in the mode names neither.
  return (
    mode.sharedWith ??
    mode.onlyReadBy ??
    markedEntries(props)[0]?.reader ??
    "other software"
  );
};

/** @returns whether the Driver can switch the slave to Synnax only. */
export const canSwitch = (props: SlaveProperties): boolean =>
  (props.settings?.mode?.switch.length ?? 0) > 0;

/** @returns whether a scan can tell that a switch to Synnax took effect. */
export const confirmsSwitch = (props: SlaveProperties): boolean =>
  markedEntries(props).length > 0;

/** @returns how the scan found the slave set up. */
export const of = (props: SlaveProperties): Setup => {
  const mode = props.settings?.mode;
  const sent = markedEntries(props).find((m) =>
    props.pdos.inputs.some((e) => sameAddress(e, m.address)),
  );
  if (sent != null && mode?.switch.length === 0)
    return { kind: "switched", reader: sent.reader };
  const onlyReader = sent?.reader ?? mode?.onlyReadBy;
  if (onlyReader != null) return { kind: "other", reader: onlyReader };
  const reader = setUpBy(props);
  if (reader != null) return { kind: "shared", reader };
  if (mode?.bus === 2) return { kind: "bus" };
  return { kind: "synnax" };
};
