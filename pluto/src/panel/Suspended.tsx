// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type panel } from "@synnaxlabs/client";
import { type PropsWithChildren, type ReactElement } from "react";

import { useEnsureRetrieved } from "@/panel/queries";
import { Scope, TabScope } from "@/panel/scope";

/** useKey resolves the active panel key from the surrounding {@link Suspended}. */
export const useKey = Scope.use;
export const useOptionalKey = Scope.useOptional;

/** useTabKey resolves the active tab key from the surrounding {@link TabProvider}. */
export const useTabKey = TabScope.use;
export const useOptionalTabKey = TabScope.useOptional;

export interface SuspendedProps extends PropsWithChildren {
  panelKey: panel.Key;
}

/**
 * Suspended retrieves the panel into the flux cache and publishes its key to
 * descendants so panel-scoped hooks resolve it without an explicit argument.
 */
export const Suspended = ({ panelKey, children }: SuspendedProps): ReactElement => {
  useEnsureRetrieved({ key: panelKey });
  return <Scope.Provider value={panelKey}>{children}</Scope.Provider>;
};

export interface TabProviderProps extends PropsWithChildren {
  tabKey: panel.TabKey;
}

/** TabProvider publishes the active tab key to descendants for tab-scoped hooks. */
export const TabProvider = ({ tabKey, children }: TabProviderProps): ReactElement => (
  <TabScope.Provider value={tabKey}>{children}</TabScope.Provider>
);
