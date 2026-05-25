// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { DisconnectedError, lineplot } from "@synnaxlabs/client";
import { Access } from "@synnaxlabs/pluto";
import { type record, uuid } from "@synnaxlabs/x";

import { type Import } from "@/import";
import { create } from "@/lineplot/layout";
import { anyStateZ } from "@/lineplot/slice";

const parseImport = (data: unknown, fallbackName: string | undefined): lineplot.New => {
  const direct = lineplot.linePlotZ.safeParse(data);
  if (direct.success) {
    const { key: _key, ...rest } = direct.data;
    return { ...rest, name: fallbackName ?? rest.name };
  }
  const legacy = anyStateZ.parse({ ...(data as record.Unknown), remoteCreated: false });
  if (legacy.pendingUpload == null)
    throw new Error("Imported line plot has no body data");
  return { ...lineplot.ZERO_NEW, ...legacy.pendingUpload, name: fallbackName ?? "" };
};

export const ingest: Import.FileIngester = async (
  data,
  { layout, placeLayout, store, client, workspaceKey },
) => {
  if (!Access.updateGranted({ id: lineplot.TYPE_ONTOLOGY_ID, store, client }))
    throw new Error("You do not have permission to import line plots");
  if (client == null) throw new DisconnectedError();
  const newPayload = parseImport(data, layout?.name);
  const created = await client.lineplots.create(workspaceKey ?? uuid.ZERO, newPayload);
  store.lineplots.set(created.key, created);
  placeLayout(
    create({ ...layout, key: created.key, name: created.name }, { remote: true }),
  );
};
