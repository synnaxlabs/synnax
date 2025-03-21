// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Instrumentation } from "@synnaxlabs/alamos";
import { createContext, type PropsWithChildren, type ReactElement, use } from "react";

import { Aether } from "@/aether";
import { alamos } from "@/alamos/aether";

export interface ContextValue {
  instrumentation: Instrumentation;
}

const Context = createContext<ContextValue>({
  instrumentation: Instrumentation.NOOP,
});

export interface ProviderProps extends PropsWithChildren, alamos.ProviderState {}

export const useInstrumentation = () => use(Context).instrumentation;

export const Provider = ({ children, ...rest }: ProviderProps): ReactElement => {
  const { path } = Aether.useUnidirectional({
    type: alamos.Provider.TYPE,
    schema: alamos.providerStateZ,
    state: rest,
  });
  return <Aether.Composite path={path}>{children}</Aether.Composite>;
};
