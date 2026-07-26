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
import { useEffect, useRef, useState } from "react";
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
 * @returns whether a full reconcile pass has completed since the last
 * return-to-cold: false until the first pass finishes, reset on client change
 * and on the epoch returning to 0 (cluster replacement).
 */
export const use = (synchronizers: Synchronizers): boolean => {
  const store = useStore<unknown, UnknownAction>();
  const client = Synnax.use();
  const [verified, setVerified] = useState(false);
  const entries: [string, Synchronizer][] = Object.entries(synchronizers).map(
    ([key, useSynchronizer]) => [key, useSynchronizer()],
  );
  // Epoch-triggered reconciles read the ref so they always see the latest
  // hook closures.
  const entriesRef = useRef(entries);
  entriesRef.current = entries;
  useEffect(() => {
    if (client == null) return;
    setVerified(false);
    const params: Params = { client, store };
    // A pass finishing after its cold reset must not count: it verified
    // against a cluster the client no longer mirrors.
    let generation = 0;
    const destructors = entriesRef.current.flatMap(
      ([, { listen }]) => listen?.(params) ?? [],
    );
    const reconcile = (): void => {
      const gen = generation;
      void (async () => {
        for (const [key, synchronizer] of entriesRef.current)
          try {
            await synchronizer.reconcile(params);
          } catch (err) {
            console.error(`${key} reconcile failed`, err);
          }
        if (gen === generation) setVerified(true);
      })();
    };
    // The host demands the change stream it verifies against: with no stream
    // the epoch never leaves 0 and the workspace never settles.
    if (client.cache.epoch > 0) reconcile();
    else
      client.cache
        .ensureStreaming()
        .catch((err: unknown) => console.error("failed to open change stream", err));
    destructors.push(
      client.cache.onEpoch((epoch) => {
        if (epoch === 0) {
          generation++;
          setVerified(false);
          return;
        }
        reconcile();
      }),
    );
    return () => {
      generation++;
      destructors.forEach((destroy) => destroy());
    };
  }, [client, store]);
  return verified;
};
