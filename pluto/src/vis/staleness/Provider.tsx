// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type CrudeTimeSpan, TimeSpan } from "@synnaxlabs/x";
import { createContext, type PropsWithChildren, type ReactElement, use } from "react";

import { Aether } from "@/aether";
import { staleness } from "@/vis/staleness/aether";

// A half second keeps the reporting error small against timeouts measured in seconds.
const DEFAULT_SWEEP_INTERVAL = TimeSpan.milliseconds(500);

export interface ProviderProps extends PropsWithChildren {
  /** How often to check sources for staleness. A bare number is read as milliseconds.
   * Bounds how late a transition can be reported; keep it well under the shortest
   * staleness timeout in use. */
  sweepInterval?: CrudeTimeSpan;
  /** Whether a stale symbol shows a badge in place of the stale color. */
  badged?: boolean;
}

const Context = createContext(false);
Context.displayName = "Staleness.Context";

/** Returns whether a stale symbol shows a badge in place of the stale color. */
export const useBadged = (): boolean => use(Context);

export const Provider = ({
  children,
  sweepInterval = DEFAULT_SWEEP_INTERVAL,
  badged = false,
}: ProviderProps): ReactElement => {
  const { path } = Aether.useUnidirectional({
    type: staleness.Provider.TYPE,
    schema: staleness.Provider.z,
    state: { sweepInterval: TimeSpan.fromMilliseconds(sweepInterval) },
  });
  return (
    <Context value={badged}>
      <Aether.Composite path={path}>{children}</Aether.Composite>
    </Context>
  );
};
