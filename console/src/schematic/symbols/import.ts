// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  DisconnectedError,
  group,
  type ontology,
  schematic,
  type Synnax as Client,
} from "@synnaxlabs/client";
import { Group, Status, Synnax } from "@synnaxlabs/pluto";
import { status, uuid } from "@synnaxlabs/x";
import { join, sep } from "@tauri-apps/api/path";
import { open } from "@tauri-apps/plugin-dialog";
import { readTextFile } from "@tauri-apps/plugin-fs";
import { useCallback } from "react";

import { Runtime } from "@/runtime";
import { groupManifestZ, SYMBOL_FILE_FILTERS } from "@/schematic/symbols/types";

const parseAndCreateSymbol = async (
  client: Client,
  filePath: string,
  parentID: ontology.ID,
): Promise<schematic.symbol.Symbol> => {
  const data = await readTextFile(filePath);
  const parsed = schematic.symbol.symbolZ.parse(JSON.parse(data));
  return await client.workspaces.schematics.symbols.create({
    ...parsed,
    key: uuid.create(),
    parent: parentID,
  });
};

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
        filters: SYMBOL_FILE_FILTERS,
        multiple: true,
        directory: false,
      });
      if (paths == null) return;
      const symbolGroup = await client.workspaces.schematics.symbols.retrieveGroup();
      const parentID = parentGroup
        ? group.ontologyID(parentGroup)
        : group.ontologyID(symbolGroup.key);

      await Promise.all(
        paths.map(async (path) => {
          try {
            const created = await parseAndCreateSymbol(client, path, parentID);
            addStatus({
              variant: "success",
              message: `Successfully imported symbol: ${created.name}`,
            });
          } catch (e) {
            const fileName = path.split(sep()).pop();
            handleError(e, `Failed to import symbol from ${fileName}`);
          }
        }),
      );
    }, "Failed to import symbols");
  }, [client, handleError, addStatus, parentGroup]);
};

export const useImportGroup = (): (() => void) => {
  const client = Synnax.use();
  const handleError = Status.useErrorHandler();
  const addStatus = Status.useAdder();
  const { updateAsync: createGroup } = Group.useCreate();

  return useCallback(() => {
    handleError(async () => {
      if (Runtime.ENGINE !== "tauri")
        throw new Error(
          "Cannot import symbol groups from a dialog when running Synnax in the browser.",
        );

      if (client == null) throw new DisconnectedError();
      const dirPath = await open({
        title: "Import Symbol Group",
        directory: true,
        multiple: false,
      });
      if (dirPath == null) return;
      const manifestPath = await join(dirPath, "manifest.json");
      const manifestData = await readTextFile(manifestPath);
      const manifest = groupManifestZ.parse(JSON.parse(manifestData));
      const symbolGroup = await client.workspaces.schematics.symbols.retrieveGroup();
      const newGroupKey = uuid.create();
      await createGroup({
        key: newGroupKey,
        name: manifest.name,
        parent: group.ontologyID(symbolGroup.key),
      });

      const parentID = group.ontologyID(newGroupKey);
      let successCount = 0;

      const errors: unknown[] = [];
      await Promise.all(
        manifest.symbols.map(async (symbolRef) => {
          try {
            const symbolPath = await join(dirPath, symbolRef.file);
            await parseAndCreateSymbol(client, symbolPath, parentID);
            successCount++;
          } catch (e) {
            errors.push(e);
          }
        }),
      );

      if (successCount === manifest.symbols.length)
        addStatus({
          variant: "success",
          message: `Successfully imported ${successCount} symbols into group "${manifest.name}"`,
        });
      else if (successCount > 0)
        addStatus({
          variant: "warning",
          message: `Imported ${successCount}/${manifest.symbols.length} symbols. Some imports failed.`,
          description: errors.map((e) => status.fromException(e).message).join("\n"),
        });
    }, "Failed to import symbol group");
  }, [client, handleError, addStatus, createGroup]);
};
