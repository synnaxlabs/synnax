// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { DisconnectedError, type Synnax as Client } from "@synnaxlabs/client";
import { useAsyncEffect, useInitializerRef } from "@synnaxlabs/lyra/hooks";
import { Flux, Synnax } from "@synnaxlabs/pluto";
import { type state } from "@synnaxlabs/x";
import { type DependencyList, useState } from "react";

/** Asks the broker for something through the Core. */
export type Request<R> = (client: Client) => Promise<R>;

export interface UseListenReturn<T extends state.State> {
  /** The latest listen. One in flight keeps the answer of the one before. */
  result: Flux.Result<T>;
  /** Listens again. */
  listen: () => void;
  /**
   * Sends a request that belongs to the current listen.
   * @returns its answer, or null once a newer listen starts or the caller unmounts.
   * @throws the error of the request, unless the answer would be null.
   */
  follow: <R>(request: Request<R>) => Promise<R | null>;
}

const OP = "listen to the broker";

/**
 * Listens to the broker on mount, when a dependency changes, and on each call to
 * `listen`. A listen is not a Flux query: nothing tells the Console when a broker
 * publishes, so only a new listen brings new data.
 */
export const useListen = <T extends state.State>(
  request: Request<T>,
  deps: DependencyList,
): UseListenReturn<T> => {
  const client = Synnax.use();
  const [result, setResult] = useState<Flux.Result<T>>(() => Flux.loadingResult(OP));
  const [round, setRound] = useState(0);
  const signalRef = useInitializerRef(() => AbortSignal.abort());
  useAsyncEffect(
    async (signal) => {
      signalRef.current = signal;
      if (client == null) {
        setResult(Flux.nullClientResult(OP));
        return;
      }
      setResult((prev) =>
        prev.variant === "loading" ? prev : Flux.loadingResult(OP, prev.data),
      );
      try {
        const data = await request(client);
        if (!signal.aborted) setResult(Flux.successResult(OP, data));
      } catch (e) {
        if (!signal.aborted) setResult(Flux.errorResult(OP, e));
      }
    },
    [client, round, ...deps],
  );
  const listen = () => setRound((r) => r + 1);
  const follow = async <R>(request: Request<R>): Promise<R | null> => {
    const signal = signalRef.current;
    if (client == null) throw new DisconnectedError();
    const answer = request(client);
    await Promise.allSettled([answer]);
    return signal.aborted ? null : await answer;
  };
  return { result, listen, follow };
};
