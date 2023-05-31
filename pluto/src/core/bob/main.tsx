import {
  PropsWithChildren,
  createContext,
  useCallback,
  useContext,
  useEffect,
} from "react";

import { useState } from "@storybook/addons";

import { WorkerMessage } from "@/core/bob/message";
import { useUniqueKey } from "@/core/hooks/useUniqueKey";
import { useTypedWorker } from "@/worker/Context";

export type BobMainSetState = (
  path: string[],
  type: string,
  state: any,
  transfer?: Transferable[]
) => void;

export interface BobContextValue {
  path: string[];
  bootstrap: (state: any, transfer?: Transferable[]) => void;
  setState: (
    path: string[],
    type: string,
    state: any,
    transfer?: Transferable[]
  ) => void;
  delete: (path: string[]) => void;
}

export const BobContext = createContext<BobContextValue>({
  path: [],
  setState: (
    path: string[],
    type: string,
    state: any,
    transfer: Transferable[] = []
  ) => {},
  delete: (path: string[]) => {},
  bootstrap: (state: any, transfer: Transferable[] = []) => {},
});

export interface UseBobComponentReturn<S extends unknown> {
  path: string[];
  state: [S, (state: S, transfer?: Transferable[]) => void];
}

export const useBobContext = (): BobContextValue => {
  const ctx = useContext(BobContext);
  if (ctx == null) throw new Error("useBobContext must be used within a BobProvider");
  return ctx;
};

export const useBobComponent = <P extends unknown>(
  type: string,
  initialState: P,
  key?: string,
  initialTransfer: Transferable[] = []
): UseBobComponentReturn<P> => {
  const oKey = useUniqueKey(key);
  const ctx = useBobContext();
  const path = [...ctx.path, oKey];
  const [state, _setState] = useState(() => {
    ctx.setState(path, type, initialState, initialTransfer);
    return initialState;
  });
  const setState = useCallback(
    (state: P, transfer: Transferable[] = []): void => {
      _setState(state);
      ctx.setState(path, type, state, transfer);
    },
    [ctx, path, type]
  );
  setState(initialState);
  useEffect(() => () => ctx.delete(path), []);
  return { path, state: [state, setState] };
};

export const useBobBootstrap = <P extends unknown>(): ((
  state: P,
  transfer?: Transferable[]
) => void) => {
  const ctx = useBobContext();
  return useCallback(
    (state: P, transfer: Transferable[] = []): void => ctx.bootstrap(state, transfer),
    [ctx]
  );
};

export interface BobProviderProps extends PropsWithChildren {
  workerKey: string;
}

export const BobProvider = ({ workerKey, children }: BobProviderProps): JSX.Element => {
  const worker = useTypedWorker<WorkerMessage>(workerKey);
  const setState = useCallback(
    (path: string[], type: string, state: any, transfer: Transferable[] = []): void =>
      worker.send({ variant: "setState", path, type, state }, transfer),
    [worker]
  );

  const delete_ = useCallback(
    (path: string[]): void => worker.send({ variant: "delete", path }),
    [worker]
  );

  const bootstrap = useCallback((state: any, transfer: Transferable[] = []): void => {
    worker.send({ variant: "bootstrap", data: state }, transfer);
  }, []);

  return (
    <BobContext.Provider value={{ path: [], setState, delete: delete_, bootstrap }}>
      {children}
    </BobContext.Provider>
  );
};

export interface BobCompositeProps extends PropsWithChildren {
  path: string[];
}

export const BobComposite = ({ children, path }: BobCompositeProps): JSX.Element => {
  const ctx = useBobContext();
  return <BobContext.Provider value={{ ...ctx, path }}>{children}</BobContext.Provider>;
};

export const Bob = {
  Provider: BobProvider,
  Composite: BobComposite,
  useComponent: useBobComponent,
  useBootstrap: useBobBootstrap,
};
