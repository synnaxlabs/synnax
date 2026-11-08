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
import { type project } from "@/project";

/**
 * Creates a throwaway project and returns its key. A panel must belong to a project or
 * a range, so tests that create panels directly put them here.
 */
export const createPanelProject = async (client: Synnax): Promise<project.Key> =>
  (await client.projects.create({ name: `project-${id.create()}` })).key;
