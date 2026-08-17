// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { DisconnectedError, group } from "@synnaxlabs/client";
import { Status, Synnax } from "@synnaxlabs/pluto";
import { useCallback } from "react";

import { Runtime } from "@/platform/runtime";

export const useImport = (): ((parentGroup: group.Key) => void) => {
  const client = Synnax.use();
  const handleError = Status.useErrorHandler();
  const addStatus = Status.useAdder();

  return useCallback(
    (parentGroup: group.Key) => {
      handleError(async () => {
        if (client == null) throw new DisconnectedError();
        const files = await Runtime.pickFiles({
          title: "Import symbol",
          filters: [{ name: "JSON", extensions: ["json"] }],
          multiple: true,
        });
        if (files == null) return;
        const parentID = group.ontologyID(parentGroup);
        await Promise.all(
          files.map(async (file) => {
            // More performant importing will come in w/ project importing.
            try {
              const data = await file.read();
              const id = await client.imex.import(data, {
                encoding: "JSON",
                fileName: file.name,
                parent: parentID,
              });
              const created = await client.schematics.symbols.retrieve({ key: id.key });
              addStatus({
                variant: "success",
                message: `Successfully imported symbol: ${created.name}`,
              });
            } catch (e) {
              handleError(e, `Failed to import symbol from ${file.name}`);
            }
          }),
        );
      }, "Failed to import symbols");
    },
    [client, handleError, addStatus],
  );
};
