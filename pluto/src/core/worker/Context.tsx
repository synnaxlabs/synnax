// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  PropsWithChildren,
  ReactElement,
  createContext,
  memo,
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useState,
} from "react";

import { TypedWorker, RoutedWorker } from "@synnaxlabs/x";

export interface WorkerContextValue {
  route: <RQ, RS = RQ>(type: string) => TypedWorker<RQ, RS>;
}

const WorkerContext = createContext<WorkerContextValue>({
  route: () => {
    throw new Error("Worker is not initialized");
  },
});

export interface WorkerProviderProps extends PropsWithChildren<{}> {
  url: URL;
  enabled?: boolean;
}

interface WorkerState {
  worker: Worker;
  router: RoutedWorker;
}

export const WorkerProvider = memo(
  ({ children, url, enabled = true }: WorkerProviderProps): ReactElement | null => {
    const [state, setState] = useState<WorkerState | null>(null);

    useEffect(() => {
      if (!enabled) return;
      const worker = new Worker(new URL(url), { type: "module" });
      const router = new RoutedWorker((e, a = []) => worker.postMessage(e, a));
      worker.onmessage = (e) => router.handle(e);
      setState({ worker, router });
      return () => worker.terminate();
    }, [url]);

    const route = useCallback(
      <RQ, RS = RQ>(type: string): TypedWorker<RQ, RS> => {
        if (state == null) throw new Error("Worker is not initialized");
        return state.router.route(type);
      },
      [state]
    );

    const value = useMemo(() => ({ route }), [route]);

    if (state == null && enabled) return null;

    return <WorkerContext.Provider value={value}>{children}</WorkerContext.Provider>;
  }
);
WorkerProvider.displayName = "WorkerProvider";

export const useTypedWorker = <RQ, RS = RQ>(
  type: string
): TypedWorker<RQ, RS> | null => {
  const ctx = useContext(WorkerContext);
  if (ctx == null) return null;
  return ctx.route(type);
};
