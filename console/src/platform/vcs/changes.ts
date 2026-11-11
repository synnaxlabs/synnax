// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type resource, type vcs } from "@synnaxlabs/client";

import { idString } from "@/platform/vcs/resource";

/**
 * Returns the uncommitted changes of the given resources. A project also takes the
 * changes of everything inside it.
 */
export const changesOf = (
  changes: vcs.Change[],
  resources: resource.ID[],
): vcs.Change[] => {
  const ids = new Set(resources.map(idString));
  return changes.filter(
    ({ resource, project }) => ids.has(resource) || ids.has(project),
  );
};
