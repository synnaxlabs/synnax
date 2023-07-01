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
  useEffect,
  useState,
} from "react";

import { Synnax, SynnaxProps, TimeSpan } from "@synnaxlabs/client";

interface ClientContextValue {
  client: Synnax | null;
}

const ClientContext = createContext<ClientContextValue>({ client: null });

export const useClient = (): Synnax | null => useContext(ClientContext).client;

export interface ClientProviderProps extends PropsWithChildren {
  connParams?: SynnaxProps;
}

export const ClientProvider = ({
  connParams,
  children,
}: ClientProviderProps): ReactElement => {
  const [state, setState] = useState<{ client: Synnax | null }>({ client: null });

  useEffect(() => {
    if (connParams == null) return;

    if (state.client != null) state.client.close();

    const client = new Synnax({
      ...connParams,
      connectivityPollFrequency: TimeSpan.seconds(5),
    });
    client.connectivity
      .check()
      .then(() => {
        if (client.connectivity.status() !== "connected") return;
        setState((c) => {
          if (c.client != null) c.client.close();
          return { client };
        });
      })
      .catch(console.error);

    return () => {
      client.close();
      setState({ client: null });
    };
  }, [connParams]);

  return <ClientContext.Provider value={state}>{children}</ClientContext.Provider>;
};
