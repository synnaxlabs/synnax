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

import { type Import } from "@/import";
import { create, LAYOUT_TYPE } from "@/schematic/layout";
import { anyStateZ } from "@/schematic/slice";

export const parseImport = (
  data: unknown,
  fallbackName: string | undefined,
): schematic.New => {
  // Legacy console-state exports are tried first because their schemas are strict: they
  // require a version literal plus the console-only editable / control / viewport /
  // props fields, so a current typed export never matches and falls through to the
  // direct branch. The typed schematicZ is the opposite — it strips unknown keys and
  // defaults configs to {}, so trying it first silently accepts a legacy file, drops
  // its props, and yields a schematic with no symbol configs (a blank import).
  if (typeof data === "object" && data != null) {
    const legacy = anyStateZ.safeParse({ ...data, remoteCreated: false });
    if (legacy.success) {
      if (legacy.data.pendingUpload == null)
        throw new Error("Imported schematic has no graph data");
      const { snapshot, nodes, edges, configs } = legacy.data.pendingUpload;
      return { name: fallbackName ?? "Schematic", snapshot, nodes, edges, configs };
    }
  }
  const { key: _key, ...rest } = schematic.schematicZ.parse(data);
  return { ...rest, name: fallbackName ?? rest.name };
};

export const ingest: Import.FileIngester = async (
  data,
  { layout, placeLayout, store, client, projectKey },
) => {
  if (!Access.updateGranted({ id: schematic.TYPE_ONTOLOGY_ID, store, client }))
    throw new Error("You do not have permission to import schematics");
  if (client == null) throw new DisconnectedError();
  const newPayload = parseImport(data, layout?.name);
  const created = await client.schematics.create(projectKey, newPayload);
  const { key, name } = created;
  store.schematics.set(key, created);
  placeLayout(create({ ...layout, key, name, type: LAYOUT_TYPE }));
  return schematic.ontologyID(key);
};
