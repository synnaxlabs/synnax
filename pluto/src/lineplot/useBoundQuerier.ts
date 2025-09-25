// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { id } from "@synnaxlabs/x";
import { type RefObject, useCallback, useRef } from "react";

import { Aether } from "@/aether";
import { lineplot } from "@/lineplot/aether";

export interface GetBoundsFn {
  (): Promise<lineplot.AxesBounds>;
}

export const useBoundQuerier = (): GetBoundsFn => {
  const promisesRef = useRef<Map<string, (bounds: lineplot.AxesBounds) => void>>(
    new Map(),
  );

  const handleAetherChange = useCallback((state: lineplot.BoundQuerierState) => {
    const { request, response } = state;
    const handle = promisesRef.current.get(request);
    if (handle == null) return;
    handle(response.bounds);
    promisesRef.current.delete(request);
  }, []);
  const [, , setState] = Aether.use({
    type: lineplot.BoundQuerier.TYPE,
    schema: lineplot.boundQuerierStateZ,
    onAetherChange: handleAetherChange,
    initialState: {
      request: "",
      response: { request: "", bounds: {} },
    },
  });

  return useCallback(
    async () =>
      await new Promise<lineplot.AxesBounds>((resolve) => {
        const key = id.create();
        promisesRef.current.set(key, resolve);
        setState((p) => ({ ...p, request: key }));
      }),
    [],
  );
};

export interface BoundsQuerierProps {
  ref: RefObject<GetBoundsFn | null>;
}

export const BoundsQuerier = ({ ref }: BoundsQuerierProps) => {
  ref.current = useBoundQuerier();
  return null;
};
