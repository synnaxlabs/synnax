import {
  PropsWithChildren,
  ReactElement,
  createContext,
  useCallback,
  useContext,
  useEffect,
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
}

interface WorkerState {
  worker: Worker;
  router: RoutedWorker;
}

export const WorkerProvider = ({
  children,
  url,
}: WorkerProviderProps): ReactElement | null => {
  const [state, setState] = useState<WorkerState | null>(null);
  useEffect(() => {
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

  if (state == null) return null;

  return (
    <WorkerContext.Provider value={{ route }}>
      {state != null && children}
    </WorkerContext.Provider>
  );
};

export const useTypedWorker = <RQ, RS = RQ>(type: string): TypedWorker<RQ, RS> => {
  const { route } = useContext(WorkerContext);
  return route(type);
};
