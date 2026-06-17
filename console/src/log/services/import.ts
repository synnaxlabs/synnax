// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { DisconnectedError, log } from "@synnaxlabs/client";
import { Access } from "@synnaxlabs/pluto";

import { type Import } from "@/import";
import { create, LAYOUT_TYPE } from "@/log/layout";

export const parseImport = (
  data: unknown,
  fallbackName: string | undefined,
): log.New => {
  const { key: _key, ...rest } = log.logZ.parse(data);
  return { ...rest, name: fallbackName ?? rest.name };
};

export const ingest: Import.FileIngester = async (
  data,
  { layout, placeLayout, store, client, projectKey },
) => {
  if (!Access.updateGranted({ id: log.TYPE_ONTOLOGY_ID, store, client }))
    throw new Error("You do not have permission to import logs");
  if (client == null) throw new DisconnectedError();
  const newPayload = parseImport(data, layout?.name);
  const created = await client.logs.create(projectKey, newPayload);
  store.logs.set(created.key, created);
  placeLayout(
    create({ ...layout, key: created.key, name: created.name, type: LAYOUT_TYPE }),
  );
  return log.ontologyID(created.key);
};
