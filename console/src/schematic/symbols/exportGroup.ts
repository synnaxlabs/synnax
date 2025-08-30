// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { DisconnectedError, group, ontology, type schematic } from "@synnaxlabs/client";
import { Status } from "@synnaxlabs/pluto";
import { join } from "@tauri-apps/api/path";
import { open } from "@tauri-apps/plugin-dialog";
import { exists, mkdir, writeTextFile } from "@tauri-apps/plugin-fs";
import { useCallback } from "react";

import { Modals } from "@/modals";
import { Runtime } from "@/runtime";
import { type ExportedSymbol } from "@/schematic/symbols/export";

export interface GroupManifest {
  version: 1;
  type: "symbol-group";
  name: string;
  symbols: Array<{
    file: string;
    key: string;
    name: string;
  }>;
}

export interface ExportGroupContext {
  client: any; // Synnax client
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
  handleError,
  addStatus,
  confirm,
}: ExportGroupContext): Promise<void> => {
  if (client == null) throw new DisconnectedError();
  
  // Get all symbols in the group
  // First, get the children of the group from the ontology
  const children = await client.ontology.retrieveChildren(group.ontologyID(groupKey));
  
  // Filter to only schematic symbols and get their keys
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
  
  // Now retrieve the actual symbol data
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
  
  if (Runtime.ENGINE !== "tauri") {
    throw new Error("Group export is only available in the desktop application");
  }
  
  // Ask user to select a parent directory
  const parentDir = await open({
    directory: true,
    title: `Select a location to export ${groupName}`,
    recursive: true,
  });
  
  if (parentDir == null) return;
  
  // Create a subdirectory for the symbol group
  const directoryName = groupName.replace(/[^a-z0-9]/gi, "_");
  const savePath = await join(parentDir, directoryName);
  
  // Check if directory exists and confirm overwrite
  if (await exists(savePath)) {
    const shouldReplace = await confirm({
      message: `A directory already exists at ${savePath}`,
      description: "Replacing will cause the old data to be deleted.",
      cancel: { label: "Cancel" },
      confirm: { label: "Replace", variant: "error" },
    });
    if (shouldReplace !== true) return;
  }
  
  // Create the directory
  await mkdir(savePath, { recursive: true });
  
  // Create manifest
  const manifest: GroupManifest = {
    version: 1,
    type: "symbol-group",
    name: groupName,
    symbols: [],
  };
  
  // Export each symbol
  for (const symbol of symbols) {
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
    
    manifest.symbols.push({
      file: fileName,
      key: symbol.key,
      name: symbol.name,
    });
  }
  
  // Write manifest
  await writeTextFile(
    await join(savePath, "manifest.json"),
    JSON.stringify(manifest, null, 2),
  );
  
  addStatus({
    variant: "success",
    message: `Exported ${symbols.length} symbols to ${savePath}`,
  });
};

export const useExportGroup = (client: any): ((groupKey: string, groupName: string) => void) => {
  const handleError = Status.useErrorHandler();
  const addStatus = Status.useAdder();
  const confirm = Modals.useConfirm();
  
  return useCallback(
    (groupKey: string, groupName: string) => {
      handleError(
        () => exportGroup({ client, groupKey, groupName, handleError, addStatus, confirm }),
        "Failed to export symbol group",
      );
    },
    [client, handleError, addStatus, confirm],
  );
};