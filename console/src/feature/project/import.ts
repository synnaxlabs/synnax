// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { DisconnectedError } from "@synnaxlabs/client";
import { Synnax } from "@synnaxlabs/pluto";
import { useCallback } from "react";

import { Import } from "@/platform/import";
import { Session } from "@/session";

/**
 * Imports a zipped project bundle through the Core and selects the created project.
 * The Core owns the bundle format — the current manifest layout and legacy LAYOUT.json
 * directories alike — and imports atomically.
 */
export const ingest: Import.BundleIngester = async (
  name,
  bundle,
  { client, store },
) => {
  if (client == null) throw new DisconnectedError();
  const imported = await client.projects.import(bundle, { fileName: name });
  store.dispatch(Session.Project.select(imported.key));
};

/** A modal that imports a project from a .zip export or its extracted folder. */
export const useImportModal = Import.createModal({
  header: "Project.Import",
  resourceName: "project",
  useOnImport: () => {
    const client = Synnax.use();
    const store = Session.useStore();
    return useCallback(
      async (bundle: Uint8Array<ArrayBuffer>, fileName: string) => {
        if (client == null) throw new DisconnectedError();
        const imported = await client.projects.import(bundle, { fileName });
        store.dispatch(Session.Project.select(imported.key));
        return imported.name;
      },
      [client, store],
    );
  },
});
