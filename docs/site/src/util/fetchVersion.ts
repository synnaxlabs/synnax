// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

const RELEASES = "https://api.github.com/repos/synnaxlabs/synnax/releases?per_page=30";

interface Release {
  tag_name: string;
  draft: boolean;
}

const STABLE = /^synnax-v(\d+\.\d+\.\d+)$/;

const resolveVersion = async (): Promise<string> => {
  const releases = (await (await fetch(RELEASES)).json()) as Release[];
  for (const { tag_name, draft } of releases) {
    const match = STABLE.exec(tag_name);
    if (!draft && match != null) return match[1];
  }
  throw new Error("no published Synnax release found");
};

// Memoized to one API call per build; the components below render across many pages.
let cached: Promise<string> | null = null;

// Skips draft releases, so buttons never point at a build that isn't published yet.
export const fetchVersion = async (): Promise<string> => {
  cached ??= resolveVersion();
  return await cached;
};
