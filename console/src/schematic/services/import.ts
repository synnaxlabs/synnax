// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { DisconnectedError, schematic } from "@synnaxlabs/client";
import { Access } from "@synnaxlabs/pluto";
import { uuid } from "@synnaxlabs/x";

import { type Import } from "@/import";
import { create, LAYOUT_TYPE } from "@/schematic/layout";
import { anyStateZ } from "@/schematic/slice";

const parseImport = (
  data: unknown,
  fallbackName: string | undefined,
): schematic.New => {
  // Modern wire format: server-schematic shape (what extract emits).
  const direct = schematic.schematicZ.safeParse(data);
  if (direct.success) {
    const { key: _key, ...rest } = direct.data;
    return { ...rest, name: fallbackName ?? rest.name };
  }
  // Legacy wire format: pre-v6 redux state. anyStateZ migrates forward to v6,
  // populating pendingUpload from the legacy graph fields. remoteCreated is a
  // local-state concept on the source machine; force it false here so the
  // migration always extracts the graph for the new server.
  const adjusted =
    typeof data === "object" && data !== null
      ? { ...(data as Record<string, unknown>), remoteCreated: false }
      : data;
  const legacy = anyStateZ.parse(adjusted);
  if (legacy.pendingUpload == null)
    throw new Error("Imported schematic has no graph data");
  const { snapshot, nodes, edges, configs } = legacy.pendingUpload;
  return { name: fallbackName ?? "Schematic", snapshot, nodes, edges, configs };
};

export const ingest: Import.FileIngester = async (
  data,
  { layout, placeLayout, store, client, workspaceKey },
) => {
  if (!Access.updateGranted({ id: schematic.TYPE_ONTOLOGY_ID, store, client }))
    throw new Error("You do not have permission to import schematics");
  if (client == null) throw new DisconnectedError();
  const newPayload = parseImport(data, layout?.name);
  const created = await client.schematics.create(workspaceKey ?? uuid.ZERO, newPayload);
  store.schematics.set(created.key, created);
  placeLayout(
    create({
      ...layout,
      key: created.key,
      name: created.name,
      type: LAYOUT_TYPE,
    }),
  );
};
