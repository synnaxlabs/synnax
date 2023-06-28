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
} from "react";

import { isObject } from "@synnaxlabs/x";

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
  transferral: Transferable[] = []
): string => {
  const key = useUniqueKey();
  const { set } = useTelemContext();
  const setRef = useRef(false);
  if (!setRef.current) {
    set(key, type, props, transferral);
    setRef.current = true;
  }

  useEffect(() => {
    console.log("setting", key, type, props, transferral);
    set(key, type, props, transferral);
  }, [props, transferral]);

  return key;
};

export interface TelemProviderProps extends PropsWithChildren<any> {}

export const TelemProvider = ({
  children,
}: TelemProviderProps): ReactElement | null => {
  const { path, setState: send } = Aether.use({ type: Telem.TYPE, schema: telemState });
  const client = useClient();

  useLayoutEffect(() => {
    if (client != null) send({ variant: "connect", props: client.props });
  }, [client]);

  const set = useCallback(
    <P extends any>(key: string, type: string, props: P, transfer?: Transferable[]) =>
      send({ variant: "set", key, type, props }, transfer),
    [send]
  );
  const remove = useCallback((key: string) => send({ variant: "remove", key }), [send]);

  return (
    <Aether.Composite path={path}>
      <TelemContext.Provider value={{ set, remove }}>{children}</TelemContext.Provider>
    </Aether.Composite>
  );
};
