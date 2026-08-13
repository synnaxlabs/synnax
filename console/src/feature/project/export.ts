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
import { Session } from "@/session";

// The Core owns membership, document serialization, file naming, and the manifest, and
// the bundle travels as an archive, so the Console streams the response straight to the
// file the user picks without ever holding it in memory.
export const useExport = (): ((key: project.Key | null) => void) => {
  const client = Synnax.use();
  const handleError = Status.useErrorHandler();
  const addStatus = Status.useAdder();
  const store = Session.useStore();
  return useCallback(
    (key: project.Key | null) => {
      let name = "project"; // default name for error message
      handleError(async () => {
        const targetKey = key ?? Session.Project.selectSelected(store.getState());
        if (client == null) throw new DisconnectedError();
        const proj = await client.projects.retrieve(targetKey);
        name = proj.name;
        await Runtime.downloadStream({
          stream: await client.projects.export(proj.key),
          name,
          extension: "zip",
          addStatus,
        });
      }, `Failed to export ${name}`);
    },
    [client, handleError, addStatus, store],
  );
};
