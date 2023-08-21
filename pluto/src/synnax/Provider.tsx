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
  useContext,
  useState,
} from "react";

import { ConnectionState, Synnax, SynnaxProps, TimeSpan } from "@synnaxlabs/client";

import { Aether } from "@/aether";
import { useAsyncEffect } from "@/hooks/useAsyncEffect";
import { synnax } from "@/synnax/aether";

const Context = createContext<synnax.ContextValue>(synnax.ZERO_CONTEXT_VALUE);

export const use = (): Synnax | null => useContext(Context).synnax;

export const useConnectionState = (): ConnectionState => useContext(Context).state;

export interface ProviderProps extends PropsWithChildren {
  connParams?: SynnaxProps;
}

export const Provider = Aether.wrap<ProviderProps>(
  synnax.Provider.TYPE,
  ({ aetherKey, connParams, children }): ReactElement => {
    const [state, setState] = useState<synnax.ContextValue>(synnax.ZERO_CONTEXT_VALUE);

    const [{ path }, , setAetherState] = Aether.use({
      aetherKey,
      type: synnax.Provider.TYPE,
      schema: synnax.Provider.stateZ,
      initialState: { props: connParams ?? null, state: null },
    });

    useAsyncEffect(async () => {
      if (state.synnax != null) state.synnax.close();
      if (connParams == null) return setState(synnax.ZERO_CONTEXT_VALUE);

      const c = new Synnax({
        ...connParams,
        connectivityPollFrequency: TimeSpan.seconds(5),
      });

      const connectivity = await c.connectivity.check();

      setState({ synnax: c, state: connectivity });

      c.connectivity.onChange((state) => setState((prev) => ({ ...prev, state })));

      setAetherState({ props: connParams, state: connectivity });

      return () => {
        c.close();
        setState(synnax.ZERO_CONTEXT_VALUE);
        setAetherState({ props: null, state: null });
      };
    }, [connParams]);

    return (
      <Context.Provider value={state}>
        <Aether.Composite path={path}>{children}</Aether.Composite>
      </Context.Provider>
    );
  }
);
