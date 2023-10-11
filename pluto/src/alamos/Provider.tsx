// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  type PropsWithChildren,
  type ReactElement,
  createContext,
  useContext,
  useEffect,
} from "react";

import { Instrumentation } from "@synnaxlabs/alamos";

import { Aether } from "@/aether";
import { alamos } from "@/alamos/aether";
import { useMemoDeepEqualProps } from "@/hooks";

export interface ContextValue {
  instrumentation: Instrumentation;
}

const Context = createContext<ContextValue>({
  instrumentation: Instrumentation.NOOP,
});

export interface ProviderProps extends PropsWithChildren, alamos.ProviderState {}

export const useInstrumentation = (): Instrumentation =>
  useContext(Context).instrumentation;

export const Provider = Aether.wrap<ProviderProps>(
  alamos.Provider.TYPE,
  ({ aetherKey, children, ...props }): ReactElement => {
    const memoProps = useMemoDeepEqualProps(props);
    const [{ path }, , setState] = Aether.use({
      aetherKey,
      type: alamos.Provider.TYPE,
      schema: alamos.providerStateZ,
      initialState: memoProps,
    });

    useEffect(() => {
      setState(memoProps);
    }, [memoProps, setState]);

    return <Aether.Composite path={path}>{children}</Aether.Composite>;
  },
);
