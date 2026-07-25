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
import { useCallback } from "react";

import { type Symbol } from "@/feature/schematic/symbol";
import { Export } from "@/platform/export";
import { Modals } from "@/platform/modals";
import { Runtime } from "@/platform/runtime";

export const extract: Export.Extractor = async (key, { client }) => {
  if (client == null) throw new DisconnectedError();
  const symbol = await client.schematics.symbols.retrieve({ key });
  return { data: JSON.stringify(symbol), name: symbol.name };
};

export const useExport = () => Export.use(extract, "symbol");

interface ExportGroupParams {
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
}: ExportGroupParams): Promise<void> => {
  if (client == null) throw new DisconnectedError();
  const children = await client.ontology.children.retrieve({ ids: group.ontologyID(key) });
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

  const directory = await Runtime.pickWritableDirectory({
    title: `Select a location to export ${name}`,
    subdirectory: Export.sanitizeFileName(name),
  });
  if (directory == null) return;
  if (directory.preExisted) {
    const shouldReplace = await confirm({
      message: `A directory already exists at ${directory.displayPath}`,
      description: "Replacing will cause the old data to be deleted.",
      cancel: { label: "Cancel" },
      confirm: { label: "Replace", variant: "error" },
    });
    if (shouldReplace !== true) return;
  }

  const manifest: Symbol.GroupManifest = {
    version: 1,
    type: "symbol_group",
    name,
    symbols: await Promise.all(
      symbols.map(async (symbol) => {
        const fileName = `${Export.sanitizeFileName(symbol.name)}_${symbol.key.slice(0, 8)}.json`;
        await directory.writeText(fileName, JSON.stringify(symbol));
        return { file: fileName, key: symbol.key, name: symbol.name };
      }),
    ),
  };

  await directory.writeText("manifest.json", JSON.stringify(manifest));

  addStatus({
    variant: "success",
    message: `Exported ${symbols.length} symbols to ${directory.displayPath}`,
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
