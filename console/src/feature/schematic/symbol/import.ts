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
  type Synnax as Client,
} from "@synnaxlabs/client";
import { Status, Synnax } from "@synnaxlabs/pluto";
import { zipSync } from "fflate";
import { useCallback } from "react";

import { Runtime } from "@/platform/runtime";

// The Core owns symbol envelope decoding, type resolution for typeless legacy files,
// legacy-version migration, file-name naming, and group parenting, so the file's bytes
// are streamed up untouched. Returns the imported symbol's name for status messages.
const importSymbolFromData = async (
  client: Client,
  data: string,
  parentID: ontology.ID,
  fileName: string,
): Promise<string> => {
  const id = await client.imex.import(data, {
    encoding: "JSON",
    fileName,
    parent: parentID,
  });
  const created = await client.schematics.symbols.retrieve({ key: id.key });
  return created.name;
};

export const useImport = (parentGroup?: string): (() => void) => {
  const client = Synnax.use();
  const handleError = Status.useErrorHandler();
  const addStatus = Status.useAdder();

  return useCallback(() => {
    handleError(async () => {
      if (client == null) throw new DisconnectedError();
      const files = await Runtime.pickFiles({
        title: "Import symbol",
        filters: [{ name: "JSON", extensions: ["json"] }],
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

  return useCallback(() => {
    handleError(async () => {
      if (client == null) throw new DisconnectedError();
      const directory = await Runtime.pickDirectory({ title: "Import symbol group" });
      if (directory == null) return;
      // The Core owns the bundle format: manifest recognition, legacy migration,
      // membership, and validation. The directory's top-level files are zipped and
      // uploaded untouched. Entries are stored uncompressed; symbol files are small.
      const encoder = new TextEncoder();
      const entries: Record<string, Uint8Array> = {};
      for (const file of directory.files) {
        if (file.path.includes("/")) continue;
        entries[file.path] = encoder.encode(await file.read());
      }
      const imported = await client.schematics.symbols.importGroup(
        zipSync(entries, { level: 0 }),
      );
      addStatus({
        variant: "success",
        message: `Successfully imported symbol group "${imported.name}"`,
      });
    }, "Failed to import symbol group");
  }, [client, handleError, addStatus]);
};
