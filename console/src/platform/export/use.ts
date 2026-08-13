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
  type ontology,
  type Synnax as Client,
} from "@synnaxlabs/client";
import { Status, Synnax } from "@synnaxlabs/pluto";
import { useCallback } from "react";
import { z } from "zod";

import { Runtime } from "@/platform/runtime";

const FILTERS: Runtime.FileFilter[] = [{ name: "JSON", extensions: ["json"] }];

const envelopeZ = z.object({ name: z.string() });

/** A serialized resource ready to be written to disk. */
export interface FileData {
  data: string;
  name: string;
}

/**
 * Streams the Core-serialized envelope for the resource identified by id and returns
 * its bytes together with the resource name promoted from the envelope. The Core owns
 * serialization and versioning, so the returned bytes are exactly the file's contents.
 */
export const fetchFileData = async (
  client: Client,
  id: ontology.ID,
): Promise<FileData> => {
  const stream = await client.imex.export(id, { encoding: "JSON" });
  const data = await new Response(stream).text();
  return { data, name: envelopeZ.parse(JSON.parse(data)).name };
};

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
          const file = await fetchFileData(client, id);
          name = file.name;
          // Response, not Blob.stream(): jsdom implements only the former.
          const stream = new Response(file.data).body;
          if (stream == null) throw new Error("failed to open envelope stream");
          await download({
            stream,
            name,
            extension: "json",
            filters: FILTERS,
          });
        },
        `Failed to export ${name ?? id.type}`,
      );
    },
    [client, handleError, download],
  );
};
