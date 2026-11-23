// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { context } from "@synnaxlabs/lyra/context";
import type { PropsWithChildren, ReactElement } from "react";

import type { Client } from "@/platform/portal/Client";

const [Context, useContext] = context.create<Client>({
  displayName: "Portal.Context",
  providerName: "Portal.Provider",
});

export interface ProviderProps extends PropsWithChildren {
  client: Client;
}

/** Gives the components below it the portal client they reach the portal through. */
export const Provider = ({ client, children }: ProviderProps): ReactElement => (
  <Context value={client}>{children}</Context>
);

/**
 * @returns the portal client of the nearest {@link Provider}.
 * @throws {Error} if no Provider is above the caller.
 */
export const use = (): Client => useContext("Portal.use");
