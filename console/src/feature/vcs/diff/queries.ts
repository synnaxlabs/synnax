// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type resource, type vcs } from "@synnaxlabs/client";
import { Flux } from "@synnaxlabs/pluto";

import { Change } from "@/feature/vcs/diff/change";

export type FilesQuery = {
  resource: string;
  before: vcs.Rev;
  after: vcs.Rev;
  /** version changes on every open, so a second open reads the files again. */
  version: number;
};

export interface Files {
  before: vcs.File[];
  after: vcs.File[];
  /**
   * resources names every resource the files reference that still exists, such as
   * channels and devices.
   */
  resources: resource.Resource[];
}

/** Returns the parsed body of the first file with the extension, or undefined. */
export const parseFile = (files: vcs.File[], extension: string): unknown => {
  const file = files.find(({ path }) => path.endsWith(extension));
  if (file == null) return undefined;
  if (extension !== ".json") return file.content;
  return JSON.parse(file.content);
};

export const { useResult: useFiles, useRefresh: useRefreshFiles } = Flux.createRetrieve<
  FilesQuery,
  Files
>({
  name: "changes",
  retrieve: async ({ client, query: { resource, before, after } }) => {
    const [prev, next] = await Promise.all([
      client.vcs.files(before, resource),
      client.vcs.files(after, resource),
    ]);
    const ids = Change.references(parseFile(prev, ".json"), parseFile(next, ".json"));
    const resources = ids.length === 0 ? [] : await client.resources.retrieve(ids);
    return { before: prev, after: next, resources };
  },
});
