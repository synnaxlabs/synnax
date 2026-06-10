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

import { type Import } from "@/import";
import { create } from "@/lineplot/layout";
import { anyStateZ } from "@/lineplot/slice";

export const parseImport = (
  data: unknown,
  fallbackName: string | undefined,
): lineplot.New => {
  // Legacy console-state exports are tried first because their schemas are strict: they
  // require a version literal plus console-only fields (viewport, selection) and a
  // wrapped axes object, so a current typed export never matches and falls through to
  // the direct branch. The typed linePlotZ is the opposite — it strips unknown keys and
  // defaults lines/rules to empty, so trying it first risks silently accepting a legacy
  // file and dropping its body.
  if (typeof data === "object" && data != null) {
    const legacy = anyStateZ.safeParse({ ...data, remoteCreated: false });
    if (legacy.success) {
      if (legacy.data.pendingUpload == null)
        throw new Error("Imported line plot has no body data");
      const { key: _key, ...body } = legacy.data.pendingUpload;
      return { ...body, name: fallbackName ?? "" };
    }
  }
  const { key: _key, ...rest } = lineplot.linePlotZ.parse(data);
  return { ...rest, name: fallbackName ?? rest.name };
};

export const ingest: Import.FileIngester = async (
  data,
  { layout, placeLayout, store, client, workspaceKey },
) => {
  if (!Access.updateGranted({ id: lineplot.TYPE_ONTOLOGY_ID, store, client }))
    throw new Error("You do not have permission to import line plots");
  if (client == null) throw new DisconnectedError();
  const newPayload = parseImport(data, layout?.name);
  const created = await client.lineplots.create(workspaceKey, newPayload);
  store.lineplots.set(created.key, created);
  placeLayout(create({ ...layout, key: created.key, name: created.name }));
};
