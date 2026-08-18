// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { DisconnectedError, type group } from "@synnaxlabs/client";
import { Status, Synnax } from "@synnaxlabs/pluto";
import { useCallback } from "react";

import { Modals } from "@/platform/modals";

export const useDeleteGroup = (): ((group: group.Group) => void) => {
  const client = Synnax.use();
  const handleError = Status.useErrorHandler();
  const addStatus = Status.useAdder();
  const confirmDelete = Modals.useConfirmDelete({
    type: "Group",
    title: "Schematic.Symbol.Group.Delete",
  });
  return useCallback(
    (g: group.Group) => {
      handleError(async () => {
        const confirmed = await confirmDelete(g);
        if (!confirmed) return;
        if (client == null) throw new DisconnectedError();
        // The Core removes the group and its symbols in one transaction, so a failure
        // leaves both untouched.
        await client.schematics.symbols.deleteGroup(g.key);
        addStatus({ variant: "success", message: `Deleted ${g.name}` });
      }, `Failed to delete ${g.name}`);
    },
    [client, handleError, addStatus, confirmDelete],
  );
};
