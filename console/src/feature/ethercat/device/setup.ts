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
 * - other: on this port, it sends data that only reader reads.
 * - restart: it is switched to Synnax only, but sends data that only reader reads
 *   until its power restarts.
 * - shared: reader also sets it up, and it sends data that Synnax reads on this port.
 */
export type Setup =
  { kind: "synnax" } | { kind: "other" | "restart" | "shared"; reader: string };

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

/** @returns the one program that reads the entry at the address, or null for none. */
export const onlyReaderOf = (props: SlaveProperties, address: Address): string | null =>
  markedEntries(props).find((m) => sameAddress(m.address, address))?.reader ?? null;

/** @returns how the scan found the slave set up. */
export const setupOf = (props: SlaveProperties): Setup => {
  const marked = markedEntries(props);
  const mode = props.settings?.mode;
  const sent = marked.find((m) =>
    props.pdos.inputs.some((e) => sameAddress(e, m.address)),
  );
  if (sent != null)
    return {
      kind: mode != null && !mode.shared ? "restart" : "other",
      reader: sent.reader,
    };
  if (mode?.shared === true)
    return { kind: "shared", reader: marked[0]?.reader ?? "other software" };
  return { kind: "synnax" };
};

/** @returns whether other software sets up the slave, so Synnax writes no settings. */
export const isShared = (props: SlaveProperties): boolean =>
  props.settings?.mode?.shared === true;
