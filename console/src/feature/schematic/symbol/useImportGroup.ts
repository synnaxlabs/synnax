// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { DisconnectedError } from "@synnaxlabs/client";
import { type UploadBody } from "@synnaxlabs/freighter";
import { Synnax } from "@synnaxlabs/pluto";
import { useCallback } from "react";

import { Import } from "@/platform/import";

/** A modal that imports a symbol group from a .zip export or its extracted folder. */
export const useImportGroup = Import.createModal({
  header: "Schematic.Symbols.Group.Import",
  resourceName: "symbol group",
  useOnImport: () => {
    const client = Synnax.use();
    return useCallback(
      async (bundle: UploadBody) => {
        if (client == null) throw new DisconnectedError();
        const imported = await client.schematics.symbols.importGroup(bundle);
        return imported.name;
      },
      [client],
    );
  },
});
