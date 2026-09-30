// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createContext, type PropsWithChildren, type ReactElement, use } from "react";

/**
 * Runs the install of a downloaded update. It must call `install` exactly once and
 * reject when `install` rejects.
 */
export type InstallMiddleware = (install: () => Promise<void>) => Promise<void>;

const Context = createContext<InstallMiddleware>(async (install) => await install());
Context.displayName = "Context";

export interface InstallProviderProps extends PropsWithChildren {
  middleware: InstallMiddleware;
}

/** Wraps the install of every update below it in `middleware`. */
export const InstallProvider = ({
  middleware,
  children,
}: InstallProviderProps): ReactElement => (
  <Context value={middleware}>{children}</Context>
);

/** @returns The middleware that runs the install of a downloaded update. */
export const useInstallMiddleware = (): InstallMiddleware => use(Context);
