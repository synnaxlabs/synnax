// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { DisconnectedError } from "@synnaxlabs/client";

import { type Import } from "@/platform/import";
import { Session } from "@/session";

/**
 * Imports a zipped project bundle through the Core and selects the created project.
 * The Core owns the bundle format — the current manifest layout and legacy LAYOUT.json
 * directories alike — and imports atomically.
 */
export const importBundle: Import.BundleImporter = async (
  name,
  bundle,
  { client, store },
) => {
  if (client == null) throw new DisconnectedError();
  const imported = await client.projects.import(bundle, { fileName: name });
  store.dispatch(Session.Project.select(imported.key));
};
