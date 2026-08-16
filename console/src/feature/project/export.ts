// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { DisconnectedError, type project } from "@synnaxlabs/client";
import { Status, Synnax } from "@synnaxlabs/pluto";
import { useCallback } from "react";

import { Runtime } from "@/platform/runtime";

// The Core owns membership, document serialization, file naming, and the manifest, and
// the bundle travels as an archive, so the Console streams the response straight to the
// file the user picks without ever holding it in memory.
export const useExport = (): ((key: project.Key) => void) => {
  const client = Synnax.use();
  const handleError = Status.useErrorHandler();
  const download = Runtime.useDownload();
  return useCallback(
    (key: project.Key) => {
      handleError(async () => {
        if (client == null) throw new DisconnectedError();
        const { name } = await client.projects.retrieve(key);
        await download({
          stream: await client.projects.export(key),
          name,
          extension: "zip",
        });
      }, "Failed to export project");
    },
    [client, handleError, download],
  );
};
