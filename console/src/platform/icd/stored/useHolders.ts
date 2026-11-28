// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import type { icd } from "@synnaxlabs/client";
import { useMemoDeepEqual } from "@synnaxlabs/lyra/memo";
import { Status } from "@synnaxlabs/lyra/status";
import { Store } from "@synnaxlabs/lyra/store";
import { ICD as PICD } from "@synnaxlabs/pluto";
import { useEffect } from "react";

/** The ICDs that a search finds to hold some messages. */
export interface Holders {
  /** Each listed ICD that holds one of the messages. */
  icds: icd.ICD[];
  /** Whether the search reads more ICDs, because a message has no ICD yet. */
  searching: boolean;
}

/**
 * @returns the listed ICDs that hold messages. Reads on through the ICDs, one page at a
 * time, until it finds each message, reads every ICD, or a read fails.
 */
export const useHolders = (messages: icd.MessageKey[]): Holders => {
  const icds = PICD.useList({ eager: true });
  const listed = Store.useItems({
    keys: icds.data,
    subscribe: icds.subscribe,
    getItem: icds.getItem,
  });
  const wanted = new Set(messages);
  const found = new Set<icd.MessageKey>();
  const holders = listed.filter(({ messages: held }) => {
    const own = held.filter(({ key }) => wanted.has(key));
    own.forEach(({ key }) => found.add(key));
    return own.length > 0;
  });
  const missing = found.size < wanted.size;
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
  return {
    icds: useMemoDeepEqual(holders),
    searching: missing && !icds.complete && icds.variant !== "error",
  };
};
