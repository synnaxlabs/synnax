// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Status as Base } from "@synnaxlabs/lyra/status";
import { deep } from "@synnaxlabs/x";
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
  // The worker keeps a short tail of what it reported, one status per key. Each status
  // is forwarded once, even when it replaces an earlier status with the same key, so
  // trimming that tail never replays one.
  const forwarded = useRef(new Map<string, Base.Status>());
  useEffect(() => {
    const next = new Map<string, Base.Status>();
    for (const stat of statuses) {
      next.set(stat.key, stat);
      if (!deep.equal(forwarded.current.get(stat.key), stat)) add(stat);
    }
    forwarded.current = next;
    if (statuses.length > WORKER_HISTORY)
      setState((s) => ({ ...s, statuses: s.statuses.slice(-WORKER_HISTORY) }));
  }, [statuses]);
  return <Aether.Composite path={path}>{children}</Aether.Composite>;
};

/**
 * Props for {@link Aggregator}. `log` prints only main-thread errors. Errors on the
 * Aether worker print through its instrumentation.
 */
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
