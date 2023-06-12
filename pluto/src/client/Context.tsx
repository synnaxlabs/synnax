import {
  PropsWithChildren,
  ReactElement,
  createContext,
  useContext,
  useState,
} from "react";

import { Synnax, SynnaxProps, TimeSpan } from "@synnaxlabs/client";

import { useAsyncEffect } from "..";

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

  useAsyncEffect(async () => {
    if (connParams == null) return;

    const client = new Synnax({
      ...connParams,
      connectivityPollFrequency: TimeSpan.seconds(5),
    });
    await client.connectivity.check();

    if (client.connectivity.status() !== "connected") return;

    setState({ client });

    return () => {
      client.close();
      setState({ client: null });
    };
  }, [connParams]);

  return <ClientContext.Provider value={state}>{children}</ClientContext.Provider>;
};
