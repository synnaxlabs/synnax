// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type group } from "@synnaxlabs/client";
import { Schematic, Status } from "@synnaxlabs/pluto";
import { useCallback } from "react";

import { Modals } from "@/platform/modals";

export const useDeleteGroup = (): ((group: group.Group) => void) => {
  const handleError = Status.useErrorHandler();
  const confirmDelete = Modals.useConfirmDelete({
    type: "Group",
    title: "Schematic.Symbol.Group.Delete",
  });
  const { update } = Schematic.Symbol.useDeleteGroup();
  return useCallback(
    (g: group.Group) => {
      handleError(async () => {
        if (!(await confirmDelete(g))) return;
        update(g.key);
      }, `Failed to delete ${g.name}`);
    },
    [handleError, confirmDelete, update],
  );
};
