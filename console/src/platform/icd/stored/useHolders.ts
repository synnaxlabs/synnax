// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { icd } from "@synnaxlabs/client";
import { useMemoDeepEqual } from "@synnaxlabs/lyra/memo";
import { Status } from "@synnaxlabs/lyra/status";
import { Store } from "@synnaxlabs/lyra/store";
import { ICD as PICD } from "@synnaxlabs/pluto";
import { useEffect } from "react";

const messagesOf = (read: Record<string, unknown>): Set<icd.MessageKey> =>
  new Set(
    Object.keys(read)
      .map((at) => at.slice(at.lastIndexOf("#") + 1))
      // A legacy key ends in a record key, not a message key.
      .filter((message) => icd.messageKeyZ.safeParse(message).success),
  );

/**
 * @returns the keys of the ICDs that hold a message that read, the read properties of
 * a device, stores channels for. Reads on through the ICDs until it finds each such
 * message or reads every ICD.
 */
export const useHolders = (read: Record<string, unknown>): icd.Key[] => {
  const icds = PICD.useList({ eager: true });
  const listed = Store.useItems({
    keys: icds.data,
    subscribe: icds.subscribe,
    getItem: icds.getItem,
  });
  const stored = messagesOf(read);
  const found = new Set<icd.MessageKey>();
  const holders = listed.filter(({ messages }) => {
    const held = messages.filter(({ key }) => stored.has(key));
    held.forEach(({ key }) => found.add(key));
    return held.length > 0;
  });
  const missing = found.size < stored.size;
  // A failed read stops the reads, and more reads nothing once the list is complete. A
  // page can answer before a render shows it loading, so each new page reads on too.
  const count = icds.data.length;
  useEffect(() => {
    if (missing && icds.variant === "success") icds.more();
  }, [missing, icds.variant, count]);
  const addStatus = Status.useAdder();
  useEffect(() => {
    if (icds.variant === "error") addStatus(icds.status);
  }, [icds.variant]);
  return useMemoDeepEqual(holders.map(({ key }) => key));
};
