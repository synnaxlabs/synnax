// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { DisconnectedError } from "@synnaxlabs/client";

import { Export } from "@/platform/export";
import { Log } from "@/platform/log";

export const VERSION = "2.0.0";

export const extract: Export.Extractor = async (key, { client }) => {
  if (client == null) throw new DisconnectedError();
  const log = await client.logs.retrieve({ key });
  return {
    data: JSON.stringify({ ...log, type: Log.LAYOUT_TYPE, version: VERSION }),
    name: log.name,
  };
};

export const useExport = () => Export.use(extract, "log");
