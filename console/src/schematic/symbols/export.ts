// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { DisconnectedError, group, type Synnax as Client } from "@synnaxlabs/client";
import { Status, Synnax } from "@synnaxlabs/pluto";
import { join } from "@tauri-apps/api/path";
import { open } from "@tauri-apps/plugin-dialog";
import { exists, mkdir, writeTextFile } from "@tauri-apps/plugin-fs";
import { useCallback } from "react";

import { Export } from "@/export";
import { Modals } from "@/modals";
import { Runtime } from "@/runtime";
import { type GroupManifest } from "@/schematic/symbols/types";

export const extract: Export.Extractor = async (key, { client }) => {
  if (client == null) throw new DisconnectedError();
  const symbol = await client.schematics.symbols.retrieve({ key });
  return { data: JSON.stringify(symbol), name: symbol.name };
};

export const useExport = () => Export.use(extract, "symbol");

interface ExportGroupArgs {
  client: Client | null;
  group: group.Group;
  handleError: Status.ErrorHandler;
  addStatus: Status.Adder;
  confirm: Modals.PromptConfirm;
}

const exportGroup = async ({
  client,
  group: { key, name },
  addStatus,
  confirm,
}: ExportGroupArgs): Promise<void> => {
  if (client == null) throw new DisconnectedError();
  const children = await client.ontology.retrieveChildren(group.ontologyID(key));
  const symbolKeys = children
    .filter((c) => c.id.type === "schematic_symbol")
    .map((c) => c.id.key);

  if (symbolKeys.length === 0)
    return addStatus({
      variant: "warning",
      message: "No symbols found in this group to export",
    });

  const symbols = await client.schematics.symbols.retrieve({
    keys: symbolKeys,
  });

  if (!symbols || symbols.length === 0)
    return addStatus({
      variant: "warning",
      message: "No symbols found in this group to export",
    });

  if (Runtime.ENGINE !== "tauri")
    throw new Error("Group export is only available in the desktop application");

  const parentDir = await open({
    directory: true,
    title: `Select a location to export ${name}`,
    recursive: true,
  });

  if (parentDir == null) return;

  const directoryName = name.replace(/[^a-z0-9]/gi, "_");
  const savePath = await join(parentDir, directoryName);

  if (await exists(savePath)) {
    const shouldReplace = await confirm({
      message: `A directory already exists at ${savePath}`,
      description: "Replacing will cause the old data to be deleted.",
      cancel: { label: "Cancel" },
      confirm: { label: "Replace", variant: "error" },
    });
    if (shouldReplace !== true) return;
  }

  await mkdir(savePath, { recursive: true });

  const manifest: GroupManifest = {
    version: 1,
    type: "symbol_group",
    name,
    symbols: await Promise.all(
      symbols.map(async (symbol) => {
        const fileName = `${symbol.name.replace(/[^a-z0-9]/gi, "_")}_${symbol.key.slice(0, 8)}.json`;
        await writeTextFile(await join(savePath, fileName), JSON.stringify(symbol));

        return {
          file: fileName,
          key: symbol.key,
          name: symbol.name,
        };
      }),
    ),
  };

  await writeTextFile(await join(savePath, "manifest.json"), JSON.stringify(manifest));

  addStatus({
    variant: "success",
    message: `Exported ${symbols.length} symbols to ${savePath}`,
  });
};

export const useExportGroup = (): ((group: group.Group) => void) => {
  const client = Synnax.use();
  const handleError = Status.useErrorHandler();
  const addStatus = Status.useAdder();
  const confirm = Modals.useConfirm();
  return useCallback(
    (group: group.Group) => {
      handleError(
        () => exportGroup({ client, group, handleError, addStatus, confirm }),
        "Failed to export symbol group",
      );
    },
    [client, handleError, addStatus, confirm],
  );
};
