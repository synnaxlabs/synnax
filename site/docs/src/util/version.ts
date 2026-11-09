// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

/** The version of the docs that follows `main`, ahead of every release. */
export const NEXT = "next";

/** The site of the newest train, which alone keeps the pages no version owns. */
export const ROOT = "https://docs.synnaxlabs.com";

const TRAIN = /^(\d+)\.(\d+)$/;

// Older trains keep their sites but leave the menu.
const LISTED_TRAINS = 2;

/**
 * Where a version stands: `latest` for the newest train, `old` for an earlier one, and
 * `next` for `main`.
 */
export type Status = "latest" | "old" | "next";

export interface Entry {
  /** A train such as `0.61`, or `next`. */
  name: string;
  /** The origin of the version's site. */
  url: string;
}

export interface Version extends Entry {
  status: Status;
  /** The versions the menu lists: `next`, then the newest trains, newest first. */
  versions: Entry[];
}

/**
 * Parses the `DOCS_VERSION` a build receives.
 * @returns the version name, or null for an unversioned build.
 * @throws {Error} if the value is neither a train such as `0.61` nor `next`.
 */
export const parse = (raw: string | undefined): string | null => {
  if (raw == null || raw === "") return null;
  if (raw !== NEXT && !TRAIN.test(raw))
    throw new Error(`DOCS_VERSION must be a train such as 0.61 or ${NEXT}, not ${raw}`);
  return raw;
};

const order = (train: string): number[] =>
  (TRAIN.exec(train) ?? []).slice(1).map(Number);

const compare = (a: string, b: string): number => {
  const [aMajor, aMinor] = order(a);
  const [bMajor, bMinor] = order(b);
  return aMajor - bMajor || aMinor - bMinor;
};

const urlOf = (name: string, newest: string): string => {
  if (name === newest) return ROOT;
  const host = name === NEXT ? NEXT : `v${name.replace(".", "-")}`;
  return `https://${host}.docs.synnaxlabs.com`;
};

/**
 * Places a version among the published trains.
 * @param trains - The trains with a stable Core release, newest first.
 * @throws {Error} if no train is published, or if `name` is a train with no stable
 * Core release.
 */
export const resolve = (name: string, trains: string[]): Version => {
  const [newest] = trains;
  if (newest == null) throw new Error("GitHub lists no stable Core release");
  if (name !== NEXT && !trains.includes(name))
    throw new Error(`Train ${name} has no stable Core release`);
  const status: Status = name === NEXT ? "next" : name === newest ? "latest" : "old";
  const listed = trains.slice(0, LISTED_TRAINS);
  if (status === "old" && !listed.includes(name)) listed.push(name);
  listed.sort((a, b) => compare(b, a));
  return {
    name,
    url: urlOf(name, newest),
    status,
    versions: [NEXT, ...listed].map((entry) => ({
      name: entry,
      url: urlOf(entry, newest),
    })),
  };
};

/**
 * Returns the origin of the pages that only the newest train keeps, such as the blog
 * and release notes: empty when the build itself serves them.
 */
export const rootOf = (version: Version | null): string =>
  version == null || version.status === "latest" ? "" : ROOT;

/** Names the search index of a version. An unversioned build reads `docs_site`. */
export const indexName = (name: string | null): string =>
  name == null ? "docs_site" : `docs_site_${name.replace(".", "_")}`;
