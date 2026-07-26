// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type UnknownAction } from "@reduxjs/toolkit";
import { Synnax } from "@synnaxlabs/pluto";
import { useEffect, useRef } from "react";
import { useStore } from "react-redux";

import {
  type Params,
  type Synchronizer,
  type Synchronizers,
} from "@/session/synchronizer/create";

/**
 * Mounts the given synchronizers in a single effect. Listeners mount before
 * the first reconcile; reconciles run sequentially in declaration order at
 * client-ready and on every epoch bump. Remounts only on client change.
 */
export const use = (synchronizers: Synchronizers): void => {
  const store = useStore<unknown, UnknownAction>();
  const client = Synnax.use();
  const entries: [string, Synchronizer][] = Object.entries(synchronizers).map(
    ([key, useSynchronizer]) => [key, useSynchronizer()],
  );
  // Epoch-triggered reconciles read the ref so they always see the latest
  // hook closures.
  const entriesRef = useRef(entries);
  entriesRef.current = entries;
  useEffect(() => {
    if (client == null) return;
    const params: Params = { client, store };
    const destructors = entriesRef.current.flatMap(
      ([, { listen }]) => listen?.(params) ?? [],
    );
    const reconcile = (): void => {
      void (async () => {
        for (const [key, synchronizer] of entriesRef.current)
          try {
            await synchronizer.reconcile(params);
          } catch (err) {
            console.error(`${key} reconcile failed`, err);
          }
      })();
    };
    if (client.cache.epoch > 0) reconcile();
    destructors.push(client.cache.onEpoch(reconcile));
    return () => destructors.forEach((destroy) => destroy());
  }, [client, store]);
};
