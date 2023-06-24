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
  useCallback,
  useContext,
  useEffect,
  useLayoutEffect,
  useRef,
  useState,
} from "react";

import { useClient } from "@/client/Context";
import { Aether } from "@/core/aether/main";
import { useUniqueKey } from "@/core/hooks/useUniqueKey";
import { Telem, telemState } from "@/telem/TelemProvider/aether";

interface TelemContextValue {
  set: <P>(key: string, type: string, props: P, transfer?: Transferable[]) => void;
  remove: (key: string) => void;
}

const TelemContext = createContext<TelemContextValue | null>(null);

const useTelemContext = (): TelemContextValue => {
  const ctx = useContext(TelemContext);
  if (ctx === null) {
    throw new Error("No telem context found");
  }
  return ctx;
};

export const useTelemSourceControl = <P extends any>(
  type: string,
  props: P,
  transferral?: Transferable[]
): string => {
  const key = useUniqueKey();
  const { set } = useTelemContext();
  const set_ = useRef(false);
  if (!set_.current) {
    set(key, type, props, transferral);
    set_.current = true;
  }
  // useLayoutEffect(() => {
  //   set(key, type, props, transferral);
  // }, []);
  // useEffect(() => {
  //   set(key, type, props, transferral);
  // }, [key]);
  return key;
};

export interface TelemProviderProps extends PropsWithChildren<any> {}

export const TelemProvider = ({
  children,
}: TelemProviderProps): ReactElement | null => {
  const [{ path }, , send] = Aether.use(Telem.TYPE, telemState, undefined);
  const client = useClient();
  const [connected, setConnected] = useState(false);

  useLayoutEffect(() => {
    if (client != null) {
      send({ variant: "connect", props: client?.props });
      setConnected(true);
    }
  }, [client]);

  const set = useCallback(
    <P extends any>(key: string, type: string, props: P, transfer?: Transferable[]) =>
      send({ variant: "set", key, type, props }, transfer),
    [send]
  );
  const remove = useCallback((key: string) => send({ variant: "remove", key }), [send]);

  if (client == null || !connected) return null;

  return (
    <Aether.Composite path={path}>
      <TelemContext.Provider value={{ set, remove }}>{children}</TelemContext.Provider>
    </Aether.Composite>
  );
};
