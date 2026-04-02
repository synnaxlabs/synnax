// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { ZodError } from "zod";

import { type FileIngesterContext, type FileIngesters } from "@/import/ingester";

export const ingestComponent = (
  data: unknown,
  fileName: string,
  fileIngesters: FileIngesters,
  ctx: FileIngesterContext,
): void => {
  let type: string | undefined;
  if (
    typeof data === "object" &&
    data != null &&
    "type" in data &&
    typeof data.type === "string"
  )
    type = data.type;
  if (type != null) {
    const ingest = fileIngesters[type];
    ingest(data, ctx);
    return;
  }
  for (const ingest of Object.values(fileIngesters))
    try {
      ingest(data, ctx);
      return;
    } catch (e) {
      if (e instanceof ZodError) continue;
      else throw e;
    }
  throw new Error(`${fileName} cannot be imported.`);
};
