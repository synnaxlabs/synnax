// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { useCombinedStateAndRef } from "@synnaxlabs/pluto";
import { type destructor } from "@synnaxlabs/x";
import type * as monacoT from "monaco-editor";
import {
  createContext,
  type PropsWithChildren,
  use,
  useCallback,
  useMemo,
  useRef,
} from "react";

import { type Extension, initializeMonaco, type Service } from "@/code/init/initialize";

export type * as Monaco from "monaco-editor";

type Monaco = typeof monacoT;

interface ContextValue {
  monaco: Monaco | null;
  requestInit: () => void;
}

const Context = createContext<ContextValue>({ monaco: null, requestInit: () => {} });
Context.displayName = "Code.Context";
export interface ProviderProps extends PropsWithChildren {
  importExtensions: Extension[];
  initServices: Service[];
}

export const Provider = ({
  children,
  importExtensions: extensions,
  initServices: services,
}: ProviderProps) => {
  const [monaco, setMonaco, monacoRef] = useCombinedStateAndRef<Monaco | null>(null);
  const destructorRef = useRef<destructor.Async>(null);
  const requestInit = useCallback(() => {
    if (monacoRef.current != null) return;
    initializeMonaco({ extensions, services })
      .then((ret) => {
        destructorRef.current = ret.destructor;
        setMonaco(ret.monaco);
      })
      .catch(console.error);
  }, []);
  const value = useMemo(() => ({ monaco, requestInit }), [monaco, requestInit]);
  return <Context value={value}>{children}</Context>;
};

export const useMonaco = () => {
  const { monaco, requestInit } = use(Context);
  requestInit();
  return monaco;
};
