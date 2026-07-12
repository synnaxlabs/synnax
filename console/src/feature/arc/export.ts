// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { arc, DisconnectedError } from "@synnaxlabs/client";

import { Export } from "@/platform/export";

export const extract: Export.Extractor = async (key, { client }) => {
  if (client == null) throw new DisconnectedError();
  const retrieved = await client.arcs.retrieve({ key });
  return {
    data: JSON.stringify({ ...retrieved, type: arc.TYPE_ONTOLOGY_ID.type }),
    name: retrieved.name,
  };
};

export const useExport = () => Export.use(extract, "arc");
