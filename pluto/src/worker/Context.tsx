// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  type PropsWithChildren,
  type ReactElement,
  createContext,
  memo,
  useContext,
  useState,
} from "react";

import { type TypedWorker, RoutedWorker, type SenderHandler } from "@synnaxlabs/x";

import { useEffectCompare } from "@/hooks";
import { useMemoCompare } from "@/memo";

export type ContextValue =
  | {
      enabled: true;
      route: <RQ, RS = RQ>(type: string) => TypedWorker<RQ, RS>;
    }
  | {
      enabled: false;
      route: null;
    };

const Context = createContext<ContextValue>({
  enabled: false,
  route: null,
});

export interface ProviderProps extends PropsWithChildren<{}> {
  url: string | URL;
  enabled?: boolean;
}

export const Provider = memo(
  ({ children, url, enabled = true }: ProviderProps): ReactElement | null => {
    const [value, setState] = useState<ContextValue>({
      route: null,
      enabled: false,
    });

    useEffectCompare(
      () => {
        if (!enabled) return;
        console.log(url);
        const worker = new Worker(url, { type: "module" });
        worker.onmessageerror = (e) => {
          console.error(e);
          console.error(e);
          console.error(JSON.stringify(e));
        };
        worker.onerror = (e) => {
          console.error(e);
          console.error(e.message);
          console.error(JSON.stringify(e));
        };
        const router = new RoutedWorker((e, a = []) => worker.postMessage(e, a));
        worker.onmessage = (e) => router.handle(e);
        setState({
          route: <RQ, RS = RQ>(type: string): TypedWorker<RQ, RS> => {
            if (value == null) throw new Error("Worker is not initialized");
            return router.route(type);
          },
          enabled: true,
        });
        return () => worker.terminate();
      },
      ([url], [prevUrl]) => url.toString() === prevUrl.toString(),
      [url],
    );

    if (enabled && value.route == null) return null;

    return <Context.Provider value={value}>{children}</Context.Provider>;
  },
);
Provider.displayName = "worker.Provider";

export const use = <RQ, RS = RQ>(type: string): SenderHandler<RQ, RS> | null => {
  const ctx = useContext(Context);
  if (!ctx.enabled) return null;
  return useMemoCompare(
    () => ctx.route(type),
    ([a], [b]) => a === b,
    [ctx.route],
  );
};
