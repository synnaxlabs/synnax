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
import { type ontology } from "@/ontology";
import { project } from "@/project";

/**
 * Creates a throwaway project and returns its ontology ID. Panel creation requires a
 * parent, so tests that mint panels directly parent them here.
 */
export const createPanelParent = async (client: Synnax): Promise<ontology.ID> => {
  const proj = await client.projects.create({
    name: `project-${id.create()}`,
    layout: {},
  });
  return project.ontologyID(proj.key);
};
