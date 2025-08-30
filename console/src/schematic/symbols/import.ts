// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { DisconnectedError, group, schematic } from "@synnaxlabs/client";
import { Status, Synnax } from "@synnaxlabs/pluto";
import { uuid } from "@synnaxlabs/x";
import { sep } from "@tauri-apps/api/path";
import { open } from "@tauri-apps/plugin-dialog";
import { readTextFile } from "@tauri-apps/plugin-fs";
import { useCallback } from "react";
import { z } from "zod";

import { Runtime } from "@/runtime";

const exportedSymbolZ = z.object({
  version: z.literal(1),
  type: z.literal("symbol"),
  symbol: schematic.symbol.symbolZ,
});

const FILTERS = [{ name: "JSON", extensions: ["json"] }];

export const useImport = (parentGroup?: string): (() => void) => {
  const client = Synnax.use();
  const handleError = Status.useErrorHandler();
  const addStatus = Status.useAdder();

  return useCallback(() => {
    handleError(async () => {
      if (Runtime.ENGINE !== "tauri")
        throw new Error(
          "Cannot import symbols from a dialog when running Synnax in the browser.",
        );
      if (client == null) throw new DisconnectedError();
      const paths = await open({
        title: "Import Symbol",
        filters: FILTERS,
        multiple: true,
        directory: false,
      });
      if (paths == null) return;
      for (const path of paths)
        try {
          const data = await readTextFile(path);
          const parsed = exportedSymbolZ.parse(JSON.parse(data));
          const symbolGroup = await client.workspaces.schematic.symbols.retrieveGroup();
          const parentID = parentGroup
            ? group.ontologyID(parentGroup)
            : group.ontologyID(symbolGroup.key);
          const created = await client.workspaces.schematic.symbols.create({
            ...parsed.symbol,
            key: uuid.create(),
            parent: parentID,
          });

          addStatus({
            variant: "success",
            message: `Successfully imported symbol: ${created.name}`,
          });
        } catch (e) {
          const fileName = path.split(sep()).pop();
          handleError(e, `Failed to import symbol from ${fileName}`);
        }
    }, "Failed to import symbols");
  }, [client, handleError, addStatus, parentGroup]);
};
