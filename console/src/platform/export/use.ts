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

import { Runtime } from "@/platform/runtime";

export interface Params {
  /** Retrieves the resource's export stream from the Core. */
  stream: (client: Client) => Promise<ReadableStream<Uint8Array>>;
  /** The resource name. It names the exported file and the failure status. */
  name: string;
  /** The extension of the exported file, without a leading dot. */
  extension: string;
}

/**
 * Returns a callback that exports a resource, streaming the Core-serialized form the
 * given fetcher retrieves to a file the user picks.
 */
export const use = (): ((params: Params) => void) => {
  const client = Synnax.use();
  const handleError = Status.useErrorHandler();
  const download = Runtime.useDownload();
  return useCallback(
    ({ stream, name, extension }: Params) => {
      handleError(async () => {
        if (client == null) throw new DisconnectedError();
        await download({ stream: await stream(client), name, extension });
      }, `Failed to export ${name}`);
    },
    [client, handleError, download],
  );
};

export interface ResourceParams {
  id: ontology.ID;
  /** The resource name. It names the exported file. */
  name: string;
}

/**
 * Returns a callback that exports the resource identified by the given ontology ID,
 * streaming its Core-serialized JSON envelope to a file the user picks.
 */
export const useResource = (): ((params: ResourceParams) => void) => {
  const export_ = use();
  return useCallback(
    ({ id, name }: ResourceParams) =>
      export_({
        stream: async (client) => await client.imex.export(id, { encoding: "JSON" }),
        name,
        extension: "json",
      }),
    [export_],
  );
};
