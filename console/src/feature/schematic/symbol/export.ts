// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { DisconnectedError, type group } from "@synnaxlabs/client";
import { Status, Synnax } from "@synnaxlabs/pluto";
import { filename } from "@synnaxlabs/x";
import { useCallback } from "react";

import { Runtime } from "@/platform/runtime";

const ARCHIVE_EXTENSION = ".zip";

export const useExportGroup = (): ((group: group.Group) => void) => {
  const client = Synnax.use();
  const handleError = Status.useErrorHandler();
  const addStatus = Status.useAdder();
  return useCallback(
    (group: group.Group) => {
      handleError(async () => {
        if (client == null) throw new DisconnectedError();
        await Runtime.downloadStream({
          stream: await client.schematics.symbols.exportGroup(group.key),
          name: filename.sanitize(group.name, ARCHIVE_EXTENSION),
          addStatus,
        });
      }, `Failed to export ${group.name}`);
    },
    [client, handleError, addStatus],
  );
};
