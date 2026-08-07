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
  type group,
  type Synnax as Client,
} from "@synnaxlabs/client";
import { Status, Synnax } from "@synnaxlabs/pluto";
import { strings } from "@synnaxlabs/x";
import { unzipSync } from "fflate";
import { useCallback } from "react";

import { Modals } from "@/platform/modals";
import { Runtime } from "@/platform/runtime";

interface ExportGroupParams {
  client: Client | null;
  group: group.Group;
  handleError: Status.ErrorHandler;
  addStatus: Status.Adder;
  confirm: Modals.PromptConfirm;
}

// The Core owns membership, symbol serialization, file naming, and the manifest. The
// bundle arrives as a zip because a directory cannot travel over the wire, so the
// Console's only job is to unpack it onto disk.
const exportGroup = async ({
  client,
  group: { key, name },
  addStatus,
  confirm,
}: ExportGroupParams): Promise<void> => {
  if (client == null) throw new DisconnectedError();
  const stream = await client.schematics.symbols.exportGroup(key);
  const bundle = unzipSync(new Uint8Array(await new Response(stream).arrayBuffer()));
  const fileNames = Object.keys(bundle);
  const symbolCount = fileNames.length - 1;
  if (symbolCount === 0)
    return addStatus({
      variant: "warning",
      message: "No symbols found in this group to export",
    });

  const directory = await Runtime.pickWritableDirectory({
    title: `Select a location to export ${name}`,
    subdirectory: strings.sanitizeFileName(name),
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

  const decoder = new TextDecoder();
  await Promise.all(
    fileNames.map(
      async (fileName) =>
        await directory.writeText(fileName, decoder.decode(bundle[fileName])),
    ),
  );

  addStatus({
    variant: "success",
    message: `Exported ${symbolCount} symbols to ${directory.displayPath}`,
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
