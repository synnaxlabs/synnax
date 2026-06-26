// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { arc, DisconnectedError } from "@synnaxlabs/client";
import { Access } from "@synnaxlabs/pluto";

import { Editor } from "@/arc/editor";
import { anyStateZ } from "@/arc/slice";
import { type Import } from "@/import";

// parseImport resolves an imported file into an Arc create payload. Legacy console-state
// exports are tried first: they carry the graph parked under pendingUpload by the state
// migration. A current export is the typed Arc itself and falls through to arcZ.
export const parseImport = (data: unknown, fallbackName?: string): arc.New => {
  if (typeof data === "object" && data != null) {
    const legacy = anyStateZ.safeParse({ ...data, remoteCreated: false });
    if (legacy.success) {
      if (legacy.data.pendingUpload == null)
        throw new Error("Imported Arc has no graph data");
      return {
        name: fallbackName ?? "Arc",
        mode: legacy.data.pendingUpload.mode ?? "graph",
        graph: legacy.data.pendingUpload.graph,
        text: legacy.data.pendingUpload.text,
      };
    }
  }
  const { key: _key, ...rest } = arc.arcZ.parse(data);
  return { ...rest, name: fallbackName ?? rest.name };
};

export const ingest: Import.FileIngester = async (
  data,
  { layout, placeLayout, store, client },
) => {
  if (!Access.updateGranted({ id: arc.TYPE_ONTOLOGY_ID, store, client }))
    throw new Error("You do not have permission to import Arc automations");
  if (client == null) throw new DisconnectedError();
  const newPayload = parseImport(data, layout?.name);
  const created = await client.arcs.create(newPayload);
  store.arcs.set(created.key, created);
  placeLayout(Editor.create({ ...layout, key: created.key, name: created.name }));
  return arc.ontologyID(created.key);
};
