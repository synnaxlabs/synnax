// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  createContext,
  type PropsWithChildren,
  type ReactElement,
  useContext,
} from "react";

import { NOOP, type Sink } from "@/platform/analytics/sink";

const Context = createContext<Sink>(NOOP);
Context.displayName = "Analytics.Context";

export interface ProviderProps extends PropsWithChildren {
  sink: Sink;
}

export const Provider = ({ sink, children }: ProviderProps): ReactElement => (
  <Context value={sink}>{children}</Context>
);

/**
 * Returns the analytics sink. It discards everything until a {@link Provider} supplies
 * one, which is how the Console build carries no analytics at all.
 */
export const use = (): Sink => useContext(Context);
