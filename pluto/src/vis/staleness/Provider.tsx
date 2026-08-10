// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type CrudeTimeSpan, TimeSpan } from "@synnaxlabs/x";
import { type PropsWithChildren, type ReactElement } from "react";

import { Aether } from "@/aether";
import { staleness } from "@/vis/staleness/aether";

// A quarter second keeps the reporting error small against timeouts measured in
// seconds.
const DEFAULT_SWEEP_INTERVAL = TimeSpan.milliseconds(250);

export interface ProviderProps extends PropsWithChildren {
  /** How often to check sources for staleness. A bare number is read as milliseconds.
   * Bounds how late a transition can be reported; keep it well under the shortest
   * staleness timeout in use. */
  sweepInterval?: CrudeTimeSpan;
}

export const Provider = ({
  children,
  sweepInterval = DEFAULT_SWEEP_INTERVAL,
}: ProviderProps): ReactElement => {
  const { path } = Aether.useUnidirectional({
    type: staleness.Provider.TYPE,
    schema: staleness.Provider.z,
    state: { sweepInterval: TimeSpan.fromMilliseconds(sweepInterval) },
  });
  return <Aether.Composite path={path}>{children}</Aether.Composite>;
};
