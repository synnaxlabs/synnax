// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { TEST_CLIENT_PROPS } from "@synnaxlabs/client";
import {
  createContext,
  type PropsWithChildren,
  type ReactElement,
  use,
  useCallback,
  useState,
} from "react";

import { Status } from "@/status";
import { status } from "@/status/aether";
import { Synnax } from "@/synnax";
import { synnax } from "@/synnax/aether";
import { createAetherProvider } from "@/testutil/Aether";

const AetherProvider = createAetherProvider({ ...synnax.REGISTRY, ...status.REGISTRY });

interface ClientConnector {
  (connected: boolean): void;
}

const Context = createContext<ClientConnector>(() => () => {});

export const useConnectToClient = () => use(Context);

export interface ProviderProps extends PropsWithChildren {
  defaultConnected?: boolean;
}

export const SynnaxProvider = ({
  defaultConnected = true,
  ...props
}: ProviderProps): ReactElement => {
  const [isConnected, setIsConnected] = useState(defaultConnected);
  const handleConnect: ClientConnector = useCallback(
    (connected: boolean) => setIsConnected(connected),
    [],
  );
  return (
    <Context value={handleConnect}>
      <AetherProvider>
        <Status.Aggregator>
          <Synnax.Provider
            {...props}
            connParams={isConnected ? TEST_CLIENT_PROPS : undefined}
          />
        </Status.Aggregator>
      </AetherProvider>
    </Context>
  );
};
