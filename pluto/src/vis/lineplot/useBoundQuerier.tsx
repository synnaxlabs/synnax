import { id } from "@synnaxlabs/x";
import { type RefObject, useCallback, useRef } from "react";

import { Aether } from "@/aether";
import { lineplot } from "@/vis/lineplot/aether";

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
