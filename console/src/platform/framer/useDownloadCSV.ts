// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { DisconnectedError, type framer } from "@synnaxlabs/client";
import { Status, Synnax } from "@synnaxlabs/pluto";

import { Runtime } from "@/platform/runtime";

export interface DownloadCSVParams extends Omit<framer.ReadRequest, "responseType"> {
  name: string;
  onDownloadStart?: () => void;
}

export const useDownloadCSV = (): ((params: DownloadCSVParams) => void) => {
  const handleError = Status.useErrorHandler();
  const client = Synnax.use();
  const download = Runtime.useDownload();
  return ({ name, onDownloadStart, ...readParams }: DownloadCSVParams) => {
    handleError(async () => {
      if (client == null) throw new DisconnectedError();
      const stream = await client.read({ ...readParams, responseType: "csv" });
      await download({ stream, name, extension: "csv", onDownloadStart });
    }, `Failed to download CSV data for ${name}`);
  };
};
