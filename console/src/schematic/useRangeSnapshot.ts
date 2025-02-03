// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { ranger, schematic } from "@synnaxlabs/client";
import { Status, Synnax } from "@synnaxlabs/pluto";
import { strings, toArray } from "@synnaxlabs/x";
import { useMutation } from "@tanstack/react-query";

import { Range } from "@/range";

interface SchematicNameAndKey extends Pick<schematic.Schematic, "key" | "name"> {}

export const useRangeSnapshot = () => {
  const handleException = Status.useExceptionHandler();
  const addStatus = Status.useAggregator();
  const rng = Range.useSelect();
  const client = Synnax.use();
  const { mutate: snapshot } = useMutation<
    void,
    Error,
    SchematicNameAndKey | SchematicNameAndKey[],
    string
  >({
    onMutate: (schematics) =>
      `${strings.naturalLanguageJoin(
        toArray(schematics).map((s) => s.name),
        "schematic",
      )} to ${rng?.name ?? "active range"}`,
    onError: (err, _, context) => handleException(err, `Failed to snapshot ${context}`),
    onSuccess: (_, __, context) =>
      addStatus({
        variant: "success",
        message: `Successfully snapshotted ${context}`,
      }),
    mutationFn: async (schematics) => {
      if (client == null) throw new Error("Server is not available");
      if (rng == null) throw new Error("No active range selected");
      const ids = await Promise.all(
        toArray(schematics).map(async (s) => {
          const newSchematic = await client.workspaces.schematic.copy(
            s.key,
            `${s.name} (Snapshot)`,
            true,
          );
          return schematic.ontologyID(newSchematic.key);
        }),
      );
      await client.ontology.addChildren(ranger.ontologyID(rng.key), ...ids);
    },
  });
  return snapshot;
};
