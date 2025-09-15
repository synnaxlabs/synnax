// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { DisconnectedError, ranger, schematic } from "@synnaxlabs/client";
import { Status, Synnax } from "@synnaxlabs/pluto";
import { array, strings } from "@synnaxlabs/x";
import { useMutation } from "@tanstack/react-query";

import { Range } from "@/range";

interface SchematicNameAndKey extends Pick<schematic.Schematic, "key" | "name"> {}

export const useRangeSnapshot = () => {
  const handleError = Status.useErrorHandler();
  const addStatus = Status.useAdder();
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
        array.toArray(schematics).map((s) => s.name),
        "schematic",
      )} to ${rng?.name ?? "active range"}`,
    onError: (err, _, context) => handleError(err, `Failed to snapshot ${context}`),
    onSuccess: (_, __, context) =>
      addStatus({
        variant: "success",
        message: `Successfully snapshotted ${context}`,
      }),
    mutationFn: async (schematics) => {
      if (client == null) throw new DisconnectedError();
      if (rng == null) throw new Error("No active range selected");
      const ids = await Promise.all(
        array.toArray(schematics).map(async (s) => {
          const newSchematic = await client.workspaces.schematics.copy({
            key: s.key,
            name: `${s.name} (Snapshot)`,
            snapshot: true,
          });
          return schematic.ontologyID(newSchematic.key);
        }),
      );
      await client.ontology.addChildren(ranger.ontologyID(rng.key), ...ids);
    },
  });
  return snapshot;
};
