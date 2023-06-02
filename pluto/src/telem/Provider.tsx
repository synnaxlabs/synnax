import { PropsWithChildren, ReactElement, useCallback } from "react";

import { TelemContext } from "./Context";
import { WorkerMessage } from "./worker";

import { useTypedWorker } from "@/worker/Context";

export interface TelemProviderProps extends PropsWithChildren<any> {}

export const TelemProvider = ({ children }: TelemProviderProps): ReactElement => {
  const w = useTypedWorker<WorkerMessage>("telem");
  const set = useCallback(
    <P extends any>(key: string, type: string, props: P, transfer?: Transferable[]) => {
      w.send({ key, type, props }, transfer);
    },
    [w]
  );
  return <TelemContext.Provider value={{ set }}>{children}</TelemContext.Provider>;
};
