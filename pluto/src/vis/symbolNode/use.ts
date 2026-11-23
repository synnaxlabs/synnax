// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { useMemoDeepEqual } from "@synnaxlabs/lyra/memo";
import { deep } from "@synnaxlabs/x";
import { useEffect } from "react";
import { useSyncExternalStoreWithSelector } from "use-sync-external-store/with-selector";
import type { z } from "zod";

import { Aether } from "@/aether";
import { symbolNode } from "@/vis/symbolNode/aether";

export interface UseProps extends Pick<
  z.input<typeof symbolNode.stateZ>,
  "sources" | "sinks" | "stalenessTimeout"
> {
  aetherKey: string;
}

export interface UseReturn extends Pick<z.infer<typeof symbolNode.stateZ>, "readings"> {
  /** set writes value to the sink of key. */
  set: (key: string, value: number) => void;
}

/** @returns the reading of each source of a symbol node, and a write to its sinks. */
export const use = ({
  aetherKey,
  sources,
  sinks,
  stalenessTimeout,
}: UseProps): UseReturn => {
  const memoProps = useMemoDeepEqual({ sources, sinks, stalenessTimeout });
  const { setState, methods, subscribe, getSnapshot } = Aether.useLifecycle({
    aetherKey,
    type: symbolNode.SymbolNode.TYPE,
    schema: symbolNode.stateZ,
    initialState: memoProps,
    methods: symbolNode.methodsZ,
  });
  // Each state update parses readings into a new object, so equal readings must not
  // render again.
  const readings = useSyncExternalStoreWithSelector(
    subscribe,
    getSnapshot,
    undefined,
    (s) => s.readings,
    deep.equal,
  );
  useEffect(() => {
    setState((state) => ({ ...state, ...memoProps }));
  }, [memoProps, setState]);
  return { set: methods.set, readings };
};
