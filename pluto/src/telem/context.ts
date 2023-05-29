import {
  PropsWithChildren,
  ReactElement,
  createContext,
  useCallback,
  useContext,
  useId,
} from "react";

import { WorkerMessage } from "./worker";

import { useTypedWorker } from "@/worker/Context";

export interface TelemContextValue {
  set: <P>(key: string, type: string, props: P, transfer?: Transferable[]) => void;
}

export const TelemContext = createContext({
  set: <P extends any>(
    key: string,
    type: string,
    props: P,
    transfer?: Transferable[]
  ) => {},
});

export interface TelemProviderProps extends PropsWithChildren<any> {}

export const TelemProvider = ({ children }: TelemProviderProps): ReactElement => {
  const w = useTypedWorker<WorkerMessage>("telem");
  const set = useCallback(
    <P extends any>(key: string, type: string, props: P, transfer?: Transferable[]) =>
      w.send({ key, type, props }, transfer),
    [w]
  );
  return <TelemContext.Provider value={{ set }}>{children}</TelemContext.Provider>;
};

export const useTelemContext = (): TelemContextValue => {
  return useContext(TelemContext);
};

export const useTelemSourceControl = <P extends any>(
  type: string,
  props: P,
  transferral?: Transferable[]
): string => {
  const key = useId();
  const { set } = useTelemContext();
  set(key, type, props, transferral);
  return key;
};
