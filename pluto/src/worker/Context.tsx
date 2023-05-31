import {
  PropsWithChildren,
  ReactElement,
  createContext,
  useCallback,
  useContext,
  useMemo,
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
}

export const WorkerProvider = ({
  children,
  url,
}: WorkerProviderProps): ReactElement => {
  const { worker, router } = useMemo(() => {
    const worker = new Worker(new URL(url), { type: "module" });
    const router = new RoutedWorker((e, a = []) => worker.postMessage(e, a));
    worker.onmessage = (e) => router.handle(e);
    return { worker, router };
  }, []);

  const route = useCallback(
    <RQ, RS = RQ>(type: string): TypedWorker<RQ, RS> => {
      if (worker === null) {
        throw new Error("Worker is not initialized");
      }
      return router.route(type);
    },
    [router]
  );

  return <WorkerContext.Provider value={{ route }}>{children}</WorkerContext.Provider>;
};

export const useTypedWorker = <RQ, RS = RQ>(type: string): TypedWorker<RQ, RS> => {
  const { route } = useContext(WorkerContext);
  return route(type);
};
