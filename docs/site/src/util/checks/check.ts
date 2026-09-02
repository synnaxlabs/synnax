// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

/** A rendered page fetched from the running docs server. */
export interface Page {
  route: string;
  html: string;
}

export interface Context {
  /** Normalized routes of every page the crawler fetched. */
  routes: Set<string>;
  /** Base URL of the running docs server. */
  baseURL: string;
  /**
   * Fetches a URL, returning null on success and a failure reason otherwise.
   * External hosts get caching, retries, and per-host serialization; the checks' own
   * server is fetched directly.
   */
  fetchOk: (url: string) => Promise<string | null>;
}

export interface Check {
  name: string;
  /** Called once per crawled page. Failures carry their own source location. */
  page?: (page: Page) => string[];
  /**
   * Called after the crawl, for cross-page validation and network fetches. Failures
   * are streamed through report as they are found; progress reports how many of the
   * check's items have been resolved so far.
   */
  finish?: (
    ctx: Context,
    report: (message: string) => void,
    progress: (done: number, total: number) => void,
  ) => Promise<void>;
}
