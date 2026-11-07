// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { id } from "@synnaxlabs/x";

import type Synnax from "@/client";
import { project } from "@/project";
import { type resource } from "@/resource";

/**
 * Creates a throwaway project and returns its resource ID. Panel creation requires a
 * parent, so tests that mint panels directly parent them here.
 */
export const createPanelParent = async (client: Synnax): Promise<resource.ID> => {
  const proj = await client.projects.create({
    name: `project-${id.create()}`,
  });
  return project.resourceID(proj.key);
};
