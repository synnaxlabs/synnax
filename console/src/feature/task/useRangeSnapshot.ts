// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { ranger } from "@synnaxlabs/client";
import { Status } from "@synnaxlabs/lyra/status";
import { type Flux, Task } from "@synnaxlabs/pluto";
import { array, strings } from "@synnaxlabs/x";

import { Range } from "@/platform/range";

export const useRangeSnapshot = () => {
  const addStatus = Status.useAdder();
  const rng = Range.useResolve();
  const { update } = Task.useCreateSnapshot({
    afterSuccess: ({
      data: { tasks },
    }: Flux.AfterSuccessParams<Task.SnapshotParams>) => {
      const names = array.toArray(tasks).map((t) => t.name);
      const joined = strings.naturalLanguageJoin(names);
      const single = names.length === 1;
      addStatus({
        variant: "success",
        name: rng?.name ?? "",
        message: single ? "Snapshotted task" : `Snapshotted ${names.length} tasks`,
        description: single
          ? `Created a snapshot of ${joined}.`
          : `Created snapshots of ${joined}.`,
      });
    },
    afterFailure: ({ status }: Flux.AfterFailureParams<Task.SnapshotParams>) =>
      addStatus({ ...status, name: rng?.name ?? "" }),
  });
  return ({ tasks }: Omit<Task.SnapshotParams, "parentID">) => {
    if (rng == null)
      return addStatus({
        variant: "error",
        message: "Failed to snapshot tasks",
        description: "No range is active.",
      });
    const parentID = ranger.ontologyID(rng.key);
    update({ tasks, parentID });
  };
};
