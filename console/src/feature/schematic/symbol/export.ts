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
import { useCallback } from "react";

import { Runtime } from "@/platform/runtime";

const ARCHIVE_EXTENSION = ".zip";

interface ExportGroupParams {
  client: Client | null;
  group: group.Group;
  addStatus: Status.Adder;
}

// The Core owns membership, symbol serialization, file naming, and the manifest, and
// the bundle travels as an archive, so the Console streams the response straight to
// the file the user picks without ever holding it in memory.
const exportGroup = async ({
  client,
  group: { key, name },
  addStatus,
}: ExportGroupParams): Promise<void> => {
  if (client == null) throw new DisconnectedError();
  await Runtime.downloadStream({
    stream: await client.schematics.symbols.exportGroup(key),
    name: strings.sanitizeFileName(name, ARCHIVE_EXTENSION),
    addStatus,
  });
};

export const useExportGroup = (): ((group: group.Group) => void) => {
  const client = Synnax.use();
  const handleError = Status.useErrorHandler();
  const addStatus = Status.useAdder();
  return useCallback(
    (group: group.Group) => {
      handleError(
        () => exportGroup({ client, group, addStatus }),
        `Failed to export ${group.name}`,
      );
    },
    [client, handleError, addStatus],
  );
};
