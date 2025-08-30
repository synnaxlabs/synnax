// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { DisconnectedError, group, schematic } from "@synnaxlabs/client";
import { Group, Status, Synnax } from "@synnaxlabs/pluto";
import { uuid } from "@synnaxlabs/x";
import { join } from "@tauri-apps/api/path";
import { open } from "@tauri-apps/plugin-dialog";
import { readTextFile } from "@tauri-apps/plugin-fs";
import { useCallback } from "react";
import { z } from "zod";

import { Runtime } from "@/runtime";

const manifestZ = z.object({
  version: z.literal(1),
  type: z.literal("symbol-group"),
  name: z.string(),
  symbols: z.array(
    z.object({
      file: z.string(),
      key: z.string(),
      name: z.string(),
    }),
  ),
});

const exportedSymbolZ = z.object({
  version: z.literal(1),
  type: z.literal("symbol"),
  symbol: schematic.symbol.symbolZ,
});

export const useImportGroup = (): (() => void) => {
  const client = Synnax.use();
  const handleError = Status.useErrorHandler();
  const addStatus = Status.useAdder();
  const { updateAsync: createGroup } = Group.create.useDirect({ params: {} });

  return useCallback(() => {
    handleError(async () => {
      if (Runtime.ENGINE !== "tauri")
        throw new Error(
          "Cannot import symbol groups from a dialog when running Synnax in the browser.",
        );
      
      if (client == null) throw new DisconnectedError();

      // Open directory picker
      const dirPath = await open({
        title: "Import Symbol Group",
        directory: true,
        multiple: false,
      });
      
      if (dirPath == null) return;

      // Read manifest
      const manifestPath = await join(dirPath, "manifest.json");
      const manifestData = await readTextFile(manifestPath);
      const manifest = manifestZ.parse(JSON.parse(manifestData));
      
      // Get the symbol group to use as parent
      const symbolGroup = await client.workspaces.schematic.symbols.retrieveGroup();
      
      // Create a new group for the imported symbols using the same API as the UI
      const newGroupKey = uuid.create();
      await createGroup({
        key: newGroupKey,
        name: manifest.name,
        parent: group.ontologyID(symbolGroup.key),
      });
      
      const parentID = group.ontologyID(newGroupKey);
      
      // Import each symbol
      let successCount = 0;
      const errors: string[] = [];
      
      for (const symbolRef of manifest.symbols) {
        try {
          const symbolPath = await join(dirPath, symbolRef.file);
          const symbolData = await readTextFile(symbolPath);
          const parsed = exportedSymbolZ.parse(JSON.parse(symbolData));
          
          // Create the symbol with a new key
          await client.workspaces.schematic.symbols.create({
            ...parsed.symbol,
            key: uuid.create(),
            parent: parentID,
          });
          
          successCount++;
        } catch (e) {
          errors.push(`Failed to import ${symbolRef.name}: ${e}`);
          console.error(`Failed to import symbol ${symbolRef.name}:`, e);
        }
      }
      
      // Report results
      if (successCount === manifest.symbols.length) {
        addStatus({
          variant: "success",
          message: `Successfully imported ${successCount} symbols into group "${manifest.name}"`,
        });
      } else if (successCount > 0) {
        addStatus({
          variant: "warning",
          message: `Imported ${successCount}/${manifest.symbols.length} symbols. Some imports failed.`,
        });
      } else {
        throw new Error("Failed to import any symbols from the group");
      }
      
      // Log any errors for debugging
      errors.forEach((error) => console.error(error));
    }, "Failed to import symbol group");
  }, [client, handleError, addStatus, createGroup]);
};