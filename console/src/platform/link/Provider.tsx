// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type resource } from "@synnaxlabs/client";
import { context } from "@synnaxlabs/lyra/context";
import { Status } from "@synnaxlabs/lyra/status";
import { Synnax } from "@synnaxlabs/pluto";
import { type PropsWithChildren, type ReactElement } from "react";

import { type Handler } from "@/platform/link/types";

/** The handler that opens a resource of each type, keyed by the type. */
export type Handlers = Partial<Record<string, Handler>>;

const [Context, useContext] = context.create<Handlers>({
  displayName: "Link.Context",
  providerName: "Link.Provider",
});

export interface ProviderProps extends PropsWithChildren {
  handlers: Handlers;
}

/** Holds the handler that opens each type of resource, for links inside the Console. */
export const Provider = ({ handlers, children }: ProviderProps): ReactElement => (
  <Context value={handlers}>{children}</Context>
);

/** Returns the handler that opens each type of resource. */
export const useHandlers = (): Handlers => useContext("Link.useHandlers");

/**
 * Returns a function that opens a resource of the type the way a link to it does, or
 * null when no link opens the type.
 */
export const useOpen = (type: resource.Type): ((key: string) => void) | null => {
  const handler = useHandlers()[type];
  const client = Synnax.use();
  const handleError = Status.useErrorHandler();
  if (handler == null || client == null) return null;
  return (key) =>
    handleError(async () => await handler({ client, key }), `Failed to open ${type}`);
};
