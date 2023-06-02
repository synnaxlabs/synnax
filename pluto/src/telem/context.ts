import { createContext, useContext, useEffect, useId } from "react";

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
  useEffect(() => {
    set(key, type, props, transferral);
  }, [key, set, type, props, transferral]);
  return key;
};
