// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { DisconnectedError, group, imex } from "@synnaxlabs/client";
import { Status, Synnax } from "@synnaxlabs/pluto";
import { useCallback } from "react";

import { Import } from "@/platform/import";
import { Runtime } from "@/platform/runtime";

export const useImport = (): ((parentGroup: group.Key) => void) => {
  const client = Synnax.use();
  const handleError = Status.useErrorHandler();
  const addStatus = Status.useAdder();
  const importBatch = Import.useImportBatch();
  return useCallback(
    (parentGroup: group.Key) => {
      handleError(async () => {
        if (client == null) throw new DisconnectedError();
        const files = await Runtime.pickFiles({
          title: "Import symbol",
          extension: "json",
          multiple: true,
        });
        if (files == null) return;
        const parent = group.ontologyID(parentGroup);
        await importBatch({
          items: files.map((file) => ({ name: file.path, readBytes: file.readBytes })),
          // The created ID is swallowed: symbols live in the symbol tree, not tabs.
          importItem: async (file) => {
            await client.imex.import(await file.readBytes(), {
              ...imex.JSON_OPTIONS,
              fileName: file.name,
              parent,
            });
          },
          onSuccess: ({ name }) =>
            addStatus({ variant: "success", message: `Imported ${name}` }),
        });
      }, "Failed to import symbols");
    },
    [client, handleError, addStatus, importBatch],
  );
};
