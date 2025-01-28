// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Instrumentation } from "@synnaxlabs/alamos";
import {
  createContext,
  type PropsWithChildren,
  type ReactElement,
  useContext,
} from "react";

import { Aether } from "@/aether";
import { alamos } from "@/alamos/aether";

export interface ContextValue {
  instrumentation: Instrumentation;
}

const Context = createContext<ContextValue>({
  instrumentation: Instrumentation.NOOP,
});

export interface ProviderProps extends PropsWithChildren, alamos.ProviderState {}

export const useInstrumentation = (): Instrumentation =>
  useContext(Context).instrumentation;

export const Provider = ({ children, ...props }: ProviderProps): ReactElement => {
  const { path } = Aether.useUnidirectional({
    type: alamos.Provider.TYPE,
    schema: alamos.providerStateZ,
    state: props,
  });
  return <Aether.Composite path={path}>{children}</Aether.Composite>;
};
