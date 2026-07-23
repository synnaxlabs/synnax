// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { context, type Icon } from "@synnaxlabs/pluto";
import { type PropsWithChildren, type ReactElement } from "react";

export interface Registry {
  getIcon: (type: string) => Icon.ReactElement;
  parseType: (type: string) => string;
}

const [Context, useContext] = context.create<Registry>({
  displayName: "Task.RegistryContext",
  providerName: "Task.RegistryProvider",
});

export const useRegistry = (): Registry => useContext("Task.useRegistry");

export interface RegistryProviderProps extends PropsWithChildren {
  registry: Registry;
}

export const RegistryProvider = ({
  registry,
  ...rest
}: RegistryProviderProps): ReactElement => <Context value={registry} {...rest} />;
