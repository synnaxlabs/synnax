// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Action } from "@reduxjs/toolkit";
import { useSyncedRef } from "@synnaxlabs/lyra/hooks";
import { Synnax } from "@synnaxlabs/pluto";
import { useEffect, useState } from "react";
import { useStore } from "react-redux";

import {
  type Callbacks,
  type Params,
  type Synchronizers,
} from "@/session/synchronizer/create";

interface Mounted<S, A extends Action> extends Callbacks<S, A> {
  name: string;
}

/**
 * Mounts the given synchronizers in a single effect. Listeners mount before the first
 * reconcile; reconciles run sequentially in array order at client-ready and on every
 * epoch bump. Remounts only on client change.
 * @returns whether a full reconcile pass has completed since the last return-to-cold:
 * false until the first pass finishes, reset on client change and on the epoch
 * returning to 0 (Core replacement).
 */
export const use = <S, A extends Action>(
  synchronizers: Synchronizers<S, A>,
): boolean => {
  "use no memo"; // Calls a hook per synchronizer; each window passes one fixed list.
  const store = useStore<S, A>();
  const client = Synnax.use();
  const [verified, setVerified] = useState(false);
  const entries: Mounted<S, A>[] = synchronizers.map(({ name, use }) => ({
    name,
    // eslint-disable-next-line synnax/react-compiler
    ...use(),
  }));
  // Epoch-triggered reconciles read the ref so they always see the latest
  // hook closures.
  const entriesRef = useSyncedRef(entries);
  useEffect(() => {
    if (client == null) return;
    setVerified(false);
    const params: Params<S, A> = { client, store };
    // A pass finishing after its cold reset must not count: it verified against a Core
    // the client no longer mirrors.
    let generation = 0;
    const destructors = entriesRef.current.flatMap(
      ({ listen }) => listen?.(params) ?? [],
    );
    const reconcile = async (): Promise<void> => {
      const gen = generation;
      // a client change, a cold reset, or a closed client ends the pass, and its
      // results, failures included, no longer describe the Core
      const ended = (): boolean =>
        gen !== generation || client.connection.status.variant === "disabled";
      for (const synchronizer of entriesRef.current) {
        if (synchronizer.reconcile == null) continue;
        if (ended()) return;
        try {
          await synchronizer.reconcile(params);
        } catch (err) {
          if (ended()) return;
          console.error(`${synchronizer.name} reconcile failed`, err);
        }
      }
      if (gen === generation) setVerified(true);
    };
    // The connection machine demands the change stream itself; the host only
    // reacts to the continuity epochs it publishes. Epoch 0 means cold: the
    // mirror was reset, so nothing verified against it still holds.
    const { connection } = client;
    let epoch = connection.status.details.epoch;
    if (epoch > 0) void reconcile();
    destructors.push(
      connection.onChange(({ details }) => {
        if (details.epoch === epoch) return;
        epoch = details.epoch;
        if (epoch === 0) {
          generation += 1;
          setVerified(false);
          return;
        }
        void reconcile();
      }),
    );
    return () => {
      generation += 1;
      destructors.forEach((destroy) => destroy());
    };
  }, [client, store]);
  return verified;
};
