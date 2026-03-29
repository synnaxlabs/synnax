// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { context } from "@synnaxlabs/pluto";
import { type destructor } from "@synnaxlabs/x";
import type * as monacoT from "monaco-editor";
import { type PropsWithChildren, useEffect, useRef, useState } from "react";

import { initializeMonaco, type Service } from "@/code/init/initialize";

export type * as Monaco from "monaco-editor";

type Monaco = typeof monacoT;

const [Context, useContext] = context.create<Monaco | null>({
  defaultValue: null,
  displayName: "Code.Context",
});

export interface ProviderProps extends PropsWithChildren {
  initServices: Service[];
}

export const Provider = ({ children, initServices: services }: ProviderProps) => {
  const [monaco, setMonaco] = useState<Monaco | null>(null);
  const destructorRef = useRef<destructor.Async>(null);

  useEffect(() => {
    initializeMonaco({ services })
      .then((ret) => {
        destructorRef.current = ret.destructor;
        setMonaco(ret.monaco);
      })
      .catch(console.error);
  }, []);

  return <Context value={monaco}>{children}</Context>;
};

export const useMonaco = (): Monaco | null => useContext();
