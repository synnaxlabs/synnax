// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { PropsWithChildren, ReactElement, createContext, useContext } from "react";

import { Instrumentation } from "@synnaxlabs/alamos";

export interface AlamosContextValue {
  instrumentation: Instrumentation;
}

export const AlamosContext = createContext<AlamosContextValue>({
  instrumentation: Instrumentation.NOOP,
});

export interface AlamosProviderProps extends PropsWithChildren {
  instrumentation?: Instrumentation;
}

export const useInstrumentation = (): Instrumentation =>
  useContext(AlamosContext).instrumentation;

export const AlamosProvider = ({
  instrumentation = Instrumentation.NOOP,
  children,
}: AlamosProviderProps): ReactElement => (
  <AlamosContext.Provider value={{ instrumentation }}>
    {children}
  </AlamosContext.Provider>
);
