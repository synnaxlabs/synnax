// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Action } from "@reduxjs/toolkit";
import { Status, Synnax } from "@synnaxlabs/pluto";
import { useEffect, useRef, useState } from "react";
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
 * Mounts the given synchronizers in a single effect. Listeners mount before
 * the first reconcile; reconciles run sequentially in array order at
 * client-ready and on every epoch bump. Remounts only on client change.
 * @returns whether a full reconcile pass has completed since the last
 * return-to-cold: false until the first pass finishes, reset on client change
 * and on the epoch returning to 0 (cluster replacement).
 */
export const use = <S, A extends Action>(
  synchronizers: Synchronizers<S, A>,
): boolean => {
  const store = useStore<S, A>();
  const client = Synnax.use();
  const handleError = Status.useErrorHandler();
  const [verified, setVerified] = useState(false);
  const entries: Mounted<S, A>[] = synchronizers.map(({ name, use }) => ({
    name,
    ...use(),
  }));
  // Epoch-triggered reconciles read the ref so they always see the latest
  // hook closures.
  const entriesRef = useRef(entries);
  entriesRef.current = entries;
  useEffect(() => {
    if (client == null) return;
    setVerified(false);
    const params: Params<S, A> = { client, store };
    // A pass finishing after its cold reset must not count: it verified
    // against a cluster the client no longer mirrors.
    let generation = 0;
    const destructors = entriesRef.current.flatMap(
      ({ listen }) => listen?.(params) ?? [],
    );
    const reconcile = (): void => {
      const gen = generation;
      handleError(async () => {
        for (const synchronizer of entriesRef.current)
          try {
            await synchronizer.reconcile(params);
          } catch (err) {
            console.error(`${synchronizer.name} reconcile failed`, err);
          }
        if (gen === generation) setVerified(true);
      });
    };
    // The connection machine demands the change stream itself; the host only
    // reacts to the continuity epochs it publishes. Epoch 0 means cold: the
    // mirror was reset, so nothing verified against it still holds.
    const { connection } = client;
    let epoch = connection.status.details.epoch;
    if (epoch > 0) reconcile();
    destructors.push(
      connection.onChange(({ details }) => {
        if (details.epoch === epoch) return;
        epoch = details.epoch;
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
