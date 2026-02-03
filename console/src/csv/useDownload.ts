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
  type framer,
  type Synnax as Client,
} from "@synnaxlabs/client";
import { Status, Synnax } from "@synnaxlabs/pluto";

import { Runtime } from "@/runtime";

export interface DownloadParams extends Omit<framer.ReadRequest, "responseType"> {
  name: string;
  onDownloadStart?: () => void;
}

export const useDownload = (): ((params: DownloadParams) => void) => {
  const handleError = Status.useErrorHandler();
  const client = Synnax.use();
  const addStatus = Status.useAdder();
  return (params: DownloadParams) => {
    const { name } = params;
    handleError(async () => {
      if (client == null) throw new DisconnectedError();
      await download({ ...params, client, addStatus });
    }, `Failed to download CSV data for ${name}`);
  };
};

interface DownloadFnParams extends DownloadParams {
  client: Client;
  addStatus: Status.Adder;
}

const download = async ({
  client,
  name,
  addStatus,
  onDownloadStart,
  ...readParams
}: DownloadFnParams): Promise<void> => {
  const stream = await client.read({ ...readParams, responseType: "csv" });
  await Runtime.downloadStream({
    stream,
    name,
    extension: "csv",
    addStatus,
    onDownloadStart,
  });
};
