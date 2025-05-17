// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { ranger, slate } from "@synnaxlabs/client";
import { Status, Synnax } from "@synnaxlabs/pluto";
import { strings, toArray } from "@synnaxlabs/x";
import { useMutation } from "@tanstack/react-query";

import { NULL_CLIENT_ERROR } from "@/errors";
import { Range } from "@/range";

interface slateNameAndKey extends Pick<slate.Slate, "key" | "name"> {}

export const useRangeSnapshot = () => {
  const handleError = Status.useErrorHandler();
  const addStatus = Status.useAdder();
  const rng = Range.useSelect();
  const client = Synnax.use();
  const { mutate: snapshot } = useMutation<
    void,
    Error,
    slateNameAndKey | slateNameAndKey[],
    string
  >({
    onMutate: (slates) =>
      `${strings.naturalLanguageJoin(
        toArray(slates).map((s) => s.name),
        "slate",
      )} to ${rng?.name ?? "active range"}`,
    onError: (err, _, context) => handleError(err, `Failed to snapshot ${context}`),
    onSuccess: (_, __, context) =>
      addStatus({
        variant: "success",
        message: `Successfully snapshotted ${context}`,
      }),
    mutationFn: async (slates) => {
      if (client == null) throw NULL_CLIENT_ERROR;
      if (rng == null) throw new Error("No active range selected");
      const ids = await Promise.all(
        toArray(slates).map(async (s) => {
          const newslate = await client.workspaces.slate.copy(
            s.key,
            `${s.name} (Snapshot)`,
            true,
          );
          return slate.ontologyID(newslate.key);
        }),
      );
      await client.ontology.addChildren(ranger.ontologyID(rng.key), ...ids);
    },
  });
  return snapshot;
};
