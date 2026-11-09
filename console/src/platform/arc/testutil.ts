// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { arc, type resource, type Synnax } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";

export const client = createTestClient();

/** Resolves the root Arcs directory, which no other directory holds. */
export const retrieveRootDirectoryID = async (client: Synnax): Promise<resource.ID> =>
  arc.directory.resourceID((await client.arcs.directories.retrieveRoot()).key);
