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

export interface Named {
  id: resource.ID;
  name: string;
}

/**
 * Returns the uncommitted changes of the given resources. A project also takes the
 * changes of everything inside it.
 */
export const changesOf = (changes: vcs.Change[], resources: Named[]): vcs.Change[] => {
  const ids = new Set(resources.map(({ id }) => idString(id)));
  const prefixes = resources
    .filter(({ id }) => id.type === "project")
    .map(({ name }) => `projects/${name}/`);
  return changes.filter(
    ({ resource, path }) =>
      ids.has(resource) || prefixes.some((prefix) => path.startsWith(prefix)),
  );
};
