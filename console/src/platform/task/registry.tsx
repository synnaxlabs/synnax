// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { context } from "@synnaxlabs/lyra/context";
import type { Icon } from "@synnaxlabs/lyra/icon";
import type { PropsWithChildren, ReactElement } from "react";

import type { PlotChannels } from "@/platform/task/plot";

export interface Registry {
  getIcon: (type: string) => Icon.ReactElement;
  parseType: (type: string) => string;
  /** Returns undefined for a task type that has no channels to plot. */
  getPlotChannels: (type: string) => PlotChannels | undefined;
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
