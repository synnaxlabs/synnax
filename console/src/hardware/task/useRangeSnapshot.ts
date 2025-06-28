// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { DisconnectedError, type ontology, ranger } from "@synnaxlabs/client";
import { Status, Synnax } from "@synnaxlabs/pluto";
import { useMutation } from "@tanstack/react-query";
import { useStore } from "react-redux";

import { Range } from "@/range";
import { type RootState } from "@/store";

interface SnapshotToRangeArgs {
  id: ontology.ID;
  name: string;
}

export const useRangeSnapshot = () => {
  const store = useStore<RootState>();
  const client = Synnax.use();
  const handleError = Status.useErrorHandler();
  return useMutation<void, Error, SnapshotToRangeArgs[]>({
    mutationFn: async (resources) => {
      const activeRange = Range.selectActiveKey(store.getState());
      if (activeRange === null) throw new Error("No active range");
      if (client == null) throw new DisconnectedError();
      const tasks = await Promise.all(
        resources.map(({ id, name }) =>
          client.hardware.tasks.copy(id.key, `${name} (Snapshot)`, true),
        ),
      );
      const otgIDs = tasks.map(({ ontologyID }) => ontologyID);
      const rangeID = ranger.ontologyID(activeRange);
      await client.ontology.addChildren(rangeID, ...otgIDs);
    },
    onError: (e: Error) => handleError(e, "Failed to create snapshot"),
  }).mutate;
};
