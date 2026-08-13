// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { DisconnectedError, type ontology } from "@synnaxlabs/client";
import { Status, Synnax } from "@synnaxlabs/pluto";
import { useCallback } from "react";

import { Runtime } from "@/platform/runtime";

/**
 * Returns a callback that exports the resource identified by the given ontology ID,
 * streaming its Core-serialized envelope to a file the user picks.
 */
export const use = (): ((id: ontology.ID) => void) => {
  const client = Synnax.use();
  const handleError = Status.useErrorHandler();
  const download = Runtime.useDownload();
  return useCallback(
    (id: ontology.ID) => {
      let name: string | undefined;
      handleError(
        async () => {
          if (client == null) throw new DisconnectedError();
          // The file is named after the resource, which the ontology resolves without
          // reading the envelope, so the export body streams straight to disk.
          ({ name } = await client.ontology.retrieve(id));
          await download({
            stream: await client.imex.export(id, { encoding: "JSON" }),
            name,
            extension: "json",
          });
        },
        `Failed to export ${name ?? id.type}`,
      );
    },
    [client, handleError, download],
  );
};
