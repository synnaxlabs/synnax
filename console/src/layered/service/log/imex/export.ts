// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { DisconnectedError, log } from "@synnaxlabs/client";
import { z } from "zod";

import { Export } from "@/export";
import { LAYOUT_TYPE } from "@/layered/service/log/layout";

const envelopeZ = z.object({ name: z.string() });

const extract: Export.Extractor = async (key, { client }) => {
  if (client == null) throw new DisconnectedError();
  const stream = await client.imex.export(log.ontologyID(key), { encoding: "JSON" });
  const data = await new Response(stream).text();
  const { name } = envelopeZ.parse(JSON.parse(data));
  return { data, name };
};

export const useExport = () => Export.use(extract, "log");

export const EXTRACTORS: Export.Extractors = { [LAYOUT_TYPE]: extract };
