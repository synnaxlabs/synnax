// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Prints the deployments that the space-separated `VERSIONS` ask for, as JSON. `REF`
// builds one version from another ref. `RELEASED` names a train that a release changed.
// `GITHUB_TOKEN` raises the GitHub API rate limit.

import { z } from "zod";

import { plan } from "@/util/deploy/plan";
import { Releases } from "@/util/releases";
import { withSites } from "@/util/version";

const COMPARE_URL = "https://api.github.com/repos/synnaxlabs/synnax/compare";
const comparisonZ = z.object({ status: z.string() });

const token = process.env.GITHUB_TOKEN;
const headers: Record<string, string> = { Accept: "application/vnd.github+json" };
if (token != null) headers.Authorization = `Bearer ${token}`;
const releases = new Releases({ token });

// The compare API answers 404 when the branch does not exist.
const holds = async (branch: string, tag: string): Promise<boolean> => {
  const url = `${COMPARE_URL}/${tag}...${branch}?per_page=1`;
  const response = await fetch(url, { headers });
  if (response.status === 404) return false;
  if (!response.ok) throw new Error(`GitHub compare API returned ${response.status}`);
  const { status } = comparisonZ.parse(await response.json());
  return status === "ahead" || status === "identical";
};

const requested = (process.env.VERSIONS ?? "")
  .split(/\s+/)
  .filter((name) => name !== "");
const optional = (value: string | undefined): string | undefined =>
  value === "" ? undefined : value;

const deployments = await plan(
  requested,
  { ref: optional(process.env.REF), released: optional(process.env.RELEASED) },
  {
    trains: async () => withSites(await releases.trains("core")),
    newest: async (train) => await releases.onTrain("core", train),
    holds,
  },
);
console.log(JSON.stringify(deployments));
