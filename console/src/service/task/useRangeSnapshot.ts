// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { ranger } from "@synnaxlabs/client";
import { type Flux, Status, Task } from "@synnaxlabs/pluto";
import { array, strings } from "@synnaxlabs/x";
import { useCallback } from "react";

import { Range } from "@/range";

export const useRangeSnapshot = () => {
  const addStatus = Status.useAdder();
  const rng = Range.useSelect();
  const buildMessage = useCallback(
    ({ tasks }: Task.SnapshotParams) =>
      `${strings.naturalLanguageJoin(
        array.toArray(tasks).map((s) => s.name),
        "schematic",
      )} to ${rng?.name ?? "active range"}`,
    [rng],
  );
  const { update } = Task.useCreateSnapshot({
    afterSuccess: useCallback(
      ({ data }: Flux.AfterSuccessParams<Task.SnapshotParams>) =>
        addStatus({
          variant: "success",
          message: `Successfully snapshotted ${buildMessage(data)}`,
        }),
      [buildMessage, addStatus],
    ),
    afterFailure: ({ status, data }: Flux.AfterFailureParams<Task.SnapshotParams>) =>
      addStatus({ ...status, message: `Failed to snapshot ${buildMessage(data)}` }),
  });
  return ({ tasks }: Omit<Task.SnapshotParams, "parentID">) => {
    if (rng == null)
      return addStatus({
        variant: "error",
        message: "Cannot snapshot schematics without an active range",
      });
    const parentID = ranger.ontologyID(rng.key);
    update({ tasks, parentID });
  };
};
