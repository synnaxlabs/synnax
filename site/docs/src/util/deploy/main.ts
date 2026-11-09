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

import { plan } from "@/util/deploy/plan";
import { Releases } from "@/util/releases";

const BRANCHES_URL = "https://api.github.com/repos/synnaxlabs/synnax/branches";

const token = process.env.GITHUB_TOKEN;
const headers: Record<string, string> = { Accept: "application/vnd.github+json" };
if (token != null) headers.Authorization = `Bearer ${token}`;
const releases = new Releases({ token });

const branched = async (train: string): Promise<boolean> => {
  const response = await fetch(`${BRANCHES_URL}/release/core-${train}`, { headers });
  if (response.status === 404) return false;
  if (!response.ok) throw new Error(`GitHub branches API returned ${response.status}`);
  return true;
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
    trains: async () => await releases.trains("core"),
    newest: async (train) => await releases.onTrain("core", train),
    branched,
  },
);
console.log(JSON.stringify(deployments));
