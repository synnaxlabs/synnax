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
  useId,
} from "react";

import { Telem, TelemState } from "./worker";

import { useClient } from "@/client/Context";
import { Aether } from "@/core/aether/main";

export interface TelemContextValue {
  set: <P>(key: string, type: string, props: P, transfer?: Transferable[]) => void;
  remove: (key: string) => void;
}

export const TelemContext = createContext<TelemContextValue | null>(null);

export const useTelemContext = (): TelemContextValue => {
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
  const key = useId();
  const { set } = useTelemContext();
  useEffect(() => {
    set(key, type, props, transferral);
  }, [key, set, type, props, transferral]);
  return key;
};

export interface TelemProviderProps extends PropsWithChildren<any> {}

export const TelemProvider = ({ children }: TelemProviderProps): ReactElement => {
  const {
    path,
    state: [, setState],
  } = Aether.use<TelemState>(Telem.TYPE, undefined);

  const client = useClient();

  useEffect(() => {
    if (client == null) return;
    client != null && setState({ variant: "connect", props: client?.props });
  }, [client]);

  const set = useCallback(
    <P extends any>(key: string, type: string, props: P, transfer?: Transferable[]) =>
      setState({ variant: "set", key, type, props }, transfer),
    [setState]
  );
  const remove = useCallback(
    (key: string) => setState({ variant: "remove", key }),
    [setState]
  );

  return (
    <Aether.Composite path={path}>
      <TelemContext.Provider value={{ set, remove }}>{children}</TelemContext.Provider>
    </Aether.Composite>
  );
};
