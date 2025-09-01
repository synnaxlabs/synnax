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
  type Synnax,
} from "@synnaxlabs/client";
import { Status } from "@synnaxlabs/pluto";
import { join } from "@tauri-apps/api/path";
import { open } from "@tauri-apps/plugin-dialog";
import { exists, mkdir, writeTextFile } from "@tauri-apps/plugin-fs";
import { useCallback } from "react";

import { Export } from "@/export";
import { Modals } from "@/modals";
import { Runtime } from "@/runtime";
import { type ExportedSymbol, type GroupManifest } from "@/schematic/symbols/types";

export const extract: Export.Extractor = async (key, { client }) => {
  if (client == null) throw new DisconnectedError();
  const symbol = await client.workspaces.schematic.symbols.retrieve({ key });
  const exportData: ExportedSymbol = {
    version: 1,
    type: "symbol",
    symbol,
  };
  return { data: JSON.stringify(exportData, null, 2), name: symbol.name };
};

export const useExport = () => Export.use(extract, "symbol");

interface ExportGroupArgs {
  client: Synnax;
  groupKey: string;
  groupName: string;
  handleError: Status.ErrorHandler;
  addStatus: Status.Adder;
  confirm: Modals.PromptConfirm;
}

const exportGroup = async ({
  client,
  groupKey,
  groupName,
  addStatus,
  confirm,
}: ExportGroupArgs): Promise<void> => {
  if (client == null) throw new DisconnectedError();

  const children = await client.ontology.retrieveChildren(group.ontologyID(groupKey));

  const symbolKeys = children
    .filter((c: ontology.Resource) => c.id.type === "schematic_symbol")
    .map((c: ontology.Resource) => c.id.key);

  if (symbolKeys.length === 0) {
    addStatus({
      variant: "warning",
      message: "No symbols found in this group to export",
    });
    return;
  }

  const symbols = await client.workspaces.schematic.symbols.retrieve({
    keys: symbolKeys,
  });

  if (!symbols || symbols.length === 0) {
    addStatus({
      variant: "warning",
      message: "No symbols found in this group to export",
    });
    return;
  }

  if (Runtime.ENGINE !== "tauri")
    throw new Error("Group export is only available in the desktop application");

  const parentDir = await open({
    directory: true,
    title: `Select a location to export ${groupName}`,
    recursive: true,
  });

  if (parentDir == null) return;

  const directoryName = groupName.replace(/[^a-z0-9]/gi, "_");
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
    type: "symbol-group",
    name: groupName,
    symbols: await Promise.all(
      symbols.map(async (symbol) => {
        const fileName = `${symbol.name.replace(/[^a-z0-9]/gi, "_")}_${symbol.key.slice(0, 8)}.json`;

        const exportedSymbol: ExportedSymbol = {
          version: 1,
          type: "symbol",
          symbol,
        };

        await writeTextFile(
          await join(savePath, fileName),
          JSON.stringify(exportedSymbol, null, 2),
        );

        return {
          file: fileName,
          key: symbol.key,
          name: symbol.name,
        };
      }),
    ),
  };

  await writeTextFile(
    await join(savePath, "manifest.json"),
    JSON.stringify(manifest, null, 2),
  );

  addStatus({
    variant: "success",
    message: `Exported ${symbols.length} symbols to ${savePath}`,
  });
};

export const useExportGroup = (
  client: any,
): ((groupKey: string, groupName: string) => void) => {
  const handleError = Status.useErrorHandler();
  const addStatus = Status.useAdder();
  const confirm = Modals.useConfirm();
  return useCallback(
    (groupKey: string, groupName: string) => {
      handleError(
        () =>
          exportGroup({ client, groupKey, groupName, handleError, addStatus, confirm }),
        "Failed to export symbol group",
      );
    },
    [client, handleError, addStatus, confirm],
  );
};
