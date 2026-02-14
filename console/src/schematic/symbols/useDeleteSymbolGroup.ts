// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { DisconnectedError, group, type ontology } from "@synnaxlabs/client";
import { Status, Synnax } from "@synnaxlabs/pluto";
import { useCallback } from "react";

import { useConfirmDelete } from "@/ontology/hooks";

export const useDeleteSymbolGroup = (): ((group: group.Group) => void) => {
  const client = Synnax.use();
  const handleError = Status.useErrorHandler();
  const addStatus = Status.useAdder();
  const confirmDelete = useConfirmDelete({ type: "Group" });
  return useCallback(
    (g: group.Group) => {
      handleError(async () => {
        const confirmed = await confirmDelete(g);
        if (!confirmed) return;
        if (client == null) throw new DisconnectedError();
        const children = await client.ontology.retrieveChildren(
          group.ontologyID(g.key),
        );
        const symbolKeys = children
          .filter((c: ontology.Resource) => c.id.type === "schematic_symbol")
          .map((c: ontology.Resource) => c.id.key);
        if (symbolKeys.length > 0) await client.schematics.symbols.delete(symbolKeys);
        await client.groups.delete(g.key);
        addStatus({
          variant: "success",
          message:
            symbolKeys.length > 0
              ? `Deleted group and ${symbolKeys.length} symbols`
              : "Deleted group",
        });
      }, "Failed to delete symbol group");
    },
    [client, handleError, addStatus, confirmDelete],
  );
};
