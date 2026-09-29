// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Status as Base } from "@synnaxlabs/lyra/status";
import { type PropsWithChildren, type ReactElement, useEffect, useRef } from "react";

import { Aether } from "@/aether";
import { status } from "@/status/aether";

const WORKER_HISTORY = 50;

const WorkerBridge = ({ children }: PropsWithChildren): ReactElement => {
  const add = Base.useAdder();
  const [{ path }, { statuses }, setState] = Aether.use({
    type: status.Aggregator.TYPE,
    schema: status.aggregatorStateZ,
    initialState: { statuses: [] },
  });
  // The worker keeps a short tail of what it reported. Only keys not seen before are
  // forwarded, so trimming that tail never replays a status.
  const seen = useRef(new Set<string>());
  useEffect(() => {
    for (const stat of statuses) {
      if (seen.current.has(stat.key)) continue;
      seen.current.add(stat.key);
      add(stat);
    }
    if (statuses.length > WORKER_HISTORY)
      setState((s) => ({ ...s, statuses: s.statuses.slice(-WORKER_HISTORY) }));
  }, [statuses]);
  return <Aether.Composite path={path}>{children}</Aether.Composite>;
};

/** Props for {@link Aggregator}. */
export interface AggregatorProps extends Base.AggregatorProps {}

/**
 * The lyra aggregator plus a bridge that forwards statuses reported on the aether
 * worker thread. Mount one near the root of the app, inside the Aether provider.
 */
export const Aggregator = ({ children, ...rest }: AggregatorProps): ReactElement => (
  <Base.Aggregator {...rest}>
    <WorkerBridge>{children}</WorkerBridge>
  </Base.Aggregator>
);
