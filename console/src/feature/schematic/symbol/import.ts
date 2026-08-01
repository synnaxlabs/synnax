// Copyright 2026 Synnax Labs, Inc.
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
  status,
  type Synnax as Client,
} from "@synnaxlabs/client";
import { Group, Status, Synnax } from "@synnaxlabs/pluto";
import { uuid } from "@synnaxlabs/x";
import { useCallback } from "react";

import { groupManifestZ, SYMBOL_FILE_FILTERS } from "@/feature/schematic/symbol/types";
import { Import } from "@/platform/import";
import { Runtime } from "@/platform/runtime";

// The Core owns symbol envelope decoding, legacy-version migration, file-name naming,
// and group parenting, so the file's bytes are streamed up nearly untouched. The type
// field is injected because symbol files never carried one and this flow knows what it
// is importing. Returns the imported symbol's name for status messages.
const importSymbolFromData = async (
  client: Client,
  data: string,
  parentID: ontology.ID,
  fileName: string,
): Promise<string> => {
  const parsed: unknown = JSON.parse(data);
  const body =
    typeof parsed === "object" && parsed != null
      ? { type: schematic.symbol.TYPE_ONTOLOGY_ID.type, ...parsed }
      : parsed;
  await client.imex.import(JSON.stringify(body), {
    encoding: "JSON",
    fileName,
    parent: parentID,
  });
  if (
    typeof parsed === "object" &&
    parsed != null &&
    "name" in parsed &&
    typeof parsed.name === "string"
  )
    return parsed.name;
  return Import.trimFileName(fileName);
};

export const useImport = (parentGroup?: string): (() => void) => {
  const client = Synnax.use();
  const handleError = Status.useErrorHandler();
  const addStatus = Status.useAdder();

  return useCallback(() => {
    handleError(async () => {
      if (client == null) throw new DisconnectedError();
      const files = await Runtime.pickFiles({
        title: "Import Symbol",
        filters: SYMBOL_FILE_FILTERS,
        multiple: true,
      });
      if (files == null) return;
      const symbolGroup = await client.schematics.symbols.retrieveGroup();
      const parentID = parentGroup
        ? group.ontologyID(parentGroup)
        : group.ontologyID(symbolGroup.key);

      await Promise.all(
        files.map(async (file) => {
          try {
            const data = await file.read();
            const name = await importSymbolFromData(client, data, parentID, file.name);
            addStatus({
              variant: "success",
              message: `Successfully imported symbol: ${name}`,
            });
          } catch (e) {
            handleError(e, `Failed to import symbol from ${file.name}`);
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
      if (client == null) throw new DisconnectedError();
      const directory = await Runtime.pickDirectory({ title: "Import Symbol Group" });
      if (directory == null) return;
      const manifestFile = directory.files.find((f) => f.path === "manifest.json");
      if (manifestFile == null)
        throw new Error("manifest.json not found in selected directory");
      const manifest = groupManifestZ.parse(JSON.parse(await manifestFile.read()));
      const symbolGroup = await client.schematics.symbols.retrieveGroup();
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
            const symbolFile = directory.files.find((f) => f.path === symbolRef.file);
            if (symbolFile == null)
              throw new Error(`Symbol file ${symbolRef.file} not found`);
            const data = await symbolFile.read();
            await importSymbolFromData(client, data, parentID, symbolFile.path);
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
