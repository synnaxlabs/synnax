// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { DisconnectedError, table } from "@synnaxlabs/client";
import { Access } from "@synnaxlabs/pluto";

import { type Import } from "@/import";
import { create, LAYOUT_TYPE } from "@/table/layout";
import { anyStateZ } from "@/table/slice";

export const parseImport = (
  data: unknown,
  fallbackName: string | undefined,
): table.New => {
  // Legacy console-state exports are tried first because their schemas are strict: they
  // require a version literal plus the console-only lastSelected / editable / layout
  // fields, so a current typed export never matches and falls through to the direct
  // branch. The typed tableZ is the opposite — it strips unknown keys and defaults rows
  // / columns / cells to empty, so trying it first silently accepts a legacy file,
  // drops its structural model, and yields a table with no rows or columns (a blank
  // import).
  if (typeof data === "object" && data != null) {
    const legacy = anyStateZ.safeParse({ ...data, remoteCreated: false });
    if (legacy.success) {
      if (legacy.data.pendingUpload == null)
        throw new Error("Imported table has no structural data");
      const { key: _key, rows, columns, cells } = legacy.data.pendingUpload;
      return { name: fallbackName ?? "Table", rows, columns, cells };
    }
  }
  const { key: _key, ...rest } = table.tableZ.parse(data);
  return { ...rest, name: fallbackName ?? rest.name };
};

export const ingest: Import.FileIngester = async (
  data,
  { layout, placeLayout, store, client, workspaceKey },
) => {
  if (!Access.updateGranted({ id: table.TYPE_ONTOLOGY_ID, store, client }))
    throw new Error("You do not have permission to import tables");
  if (client == null) throw new DisconnectedError();
  const newPayload = parseImport(data, layout?.name);
  const created = await client.tables.create(workspaceKey, newPayload);
  store.tables.set(created.key, created);
  placeLayout(
    create({ ...layout, key: created.key, name: created.name, type: LAYOUT_TYPE }),
  );
};
