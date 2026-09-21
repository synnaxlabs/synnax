// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { TimeSpan } from "@synnaxlabs/x";

export type Product = "console" | "core" | "driver";
export type Channel = "stable" | "next";

const REPO = "synnaxlabs/synnax";
const API_URL = `https://api.github.com/repos/${REPO}/releases?per_page=100`;
const MAX_PAGES = 20;
const NEXT_LINK = /<([^>]+)>;\s*rel="next"/;
const TAG = /^(console|core|driver)\/v(\d+)\.(\d+)\.(\d+)(?:-rc\.(\d+))?$/;

/** Builds the tag a product release carries, such as `console/v0.59.0`. */
export const tag = (product: Product, version: string): string =>
  `${product}/v${version}`;

/** Builds the download URL of an asset on a product release. */
export const assetURL = (product: Product, version: string, asset: string): string =>
  `https://github.com/${REPO}/releases/download/${tag(product, version)}/${asset}`;

/** Builds the URL of the Console updater manifest on a Console release. */
export const manifestURL = (version: string): string =>
  assetURL("console", version, "latest.json");

interface Parsed {
  product: Product;
  version: string;
  order: number[];
  candidate: boolean;
}

const parse = (tagName: string): Parsed | null => {
  const match = TAG.exec(tagName);
  if (match == null) return null;
  const [, product, major, minor, patch, rc] = match;
  const candidate = rc != null;
  // A stable release outranks every candidate for its version.
  const rank = candidate ? Number(rc) : Infinity;
  return {
    product: product as Product,
    version: tagName.slice(product.length + 2),
    order: [Number(major), Number(minor), Number(patch), rank],
    candidate,
  };
};

const compare = (a: number[], b: number[]): number => {
  for (let i = 0; i < a.length; i++) if (a[i] !== b[i]) return a[i] - b[i];
  return 0;
};

/** A release as the GitHub API lists it, reduced to the fields the lookup reads. */
export interface Release {
  tag_name: string;
  draft: boolean;
  prerelease: boolean;
}

/**
 * Finds the highest version among the listed releases of a product. Semver order,
 * never release date, so a hotfix on an old train never outranks the current one.
 * @param channel - `stable` skips candidates; `next` ranks candidates with stables.
 * @returns The version without the tag prefix, such as `0.59.0`, or null if the
 * product has no published release on the channel.
 */
export const highest = (
  releases: Release[],
  product: Product,
  channel: Channel,
): string | null => {
  let best: Parsed | null = null;
  for (const release of releases) {
    if (release.draft) continue;
    const parsed = parse(release.tag_name);
    if (parsed == null || parsed.product !== product) continue;
    if (channel === "stable" && (parsed.candidate || release.prerelease)) continue;
    if (best == null || compare(parsed.order, best.order) > 0) best = parsed;
  }
  return best?.version ?? null;
};

export interface Options {
  /** Raises the GitHub API rate limit. Unauthenticated lookups still work. */
  token?: string;
  /** How long one listing serves lookups. Defaults to five minutes. */
  ttl?: TimeSpan;
  fetch?: typeof fetch;
  now?: () => number;
}

interface Listing {
  releases: Promise<Release[]>;
  expires: number;
}

/** Looks up the latest product versions from GitHub releases, caching the listing. */
export class Releases {
  private readonly token?: string;
  private readonly ttl: number;
  private readonly fetcher: typeof fetch;
  private readonly now: () => number;
  private listing: Listing | null = null;

  constructor({
    token,
    ttl = TimeSpan.minutes(5),
    fetch: fetcher = globalThis.fetch,
    now = Date.now,
  }: Options = {}) {
    this.token = token;
    this.ttl = ttl.milliseconds;
    this.fetcher = fetcher;
    this.now = now;
  }

  /**
   * Returns the highest published version of a product.
   * @param channel - `stable` (default) or `next`, which includes candidates.
   * @throws {Error} if the API is unreachable or lists no such release.
   */
  async latest(product: Product, channel: Channel = "stable"): Promise<string> {
    const version = highest(await this.list(), product, channel);
    if (version == null)
      throw new Error(`GitHub lists no ${channel} ${product} release`);
    return version;
  }

  private list(): Promise<Release[]> {
    const now = this.now();
    if (this.listing != null && this.listing.expires > now)
      return this.listing.releases;
    const releases = this.fetchListing();
    const listing = { releases, expires: now + this.ttl };
    this.listing = listing;
    // A failed listing is not cached, so the next lookup retries.
    void releases.catch(() => {
      if (this.listing === listing) this.listing = null;
    });
    return releases;
  }

  // GitHub lists by creation date, and every product release is newer than every
  // legacy one, so the walk stops at the first page without a product release.
  private async fetchListing(): Promise<Release[]> {
    const headers: Record<string, string> = { Accept: "application/vnd.github+json" };
    if (this.token != null) headers.Authorization = `Bearer ${this.token}`;
    const releases: Release[] = [];
    let url: string | null = API_URL;
    for (let page = 0; url != null && page < MAX_PAGES; page++) {
      const response: Response = await this.fetcher(url, { headers });
      if (!response.ok)
        throw new Error(`GitHub releases API returned ${response.status}`);
      const body: unknown = await response.json();
      if (!Array.isArray(body)) throw new Error("GitHub releases API returned no list");
      const listed = body as Release[];
      releases.push(...listed);
      if (!listed.some(({ tag_name }) => parse(tag_name) != null)) break;
      url = NEXT_LINK.exec(response.headers.get("link") ?? "")?.[1] ?? null;
    }
    return releases;
  }
}
