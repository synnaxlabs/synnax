// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { arc } from "@synnaxlabs/client";
import { Access } from "@synnaxlabs/pluto";

import { Editor } from "@/arc/editor";
import { anyStateZ } from "@/arc/slice";
import { type Import } from "@/import";

export const ingest: Import.FileIngester = (
  data,
  { layout, placeLayout, store, client },
) => {
  const state = anyStateZ.parse(data);
  if (!Access.updateGranted({ id: arc.TYPE_ONTOLOGY_ID, store, client }))
    throw new Error("You do not have permission to import Arc automations");
  placeLayout(Editor.create({ ...state, key: layout?.key, ...layout }));
};
