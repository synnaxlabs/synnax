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

/** A line of a source file. The file is relative to the docs package. */
export interface Source {
  file: string;
  line: number;
}

export interface Context {
  /** Normalized routes of every page the crawler fetched. */
  routes: Set<string>;
  /** Base URL of the running docs server. */
  baseURL: string;
  /**
   * Fetches a URL, returning null on success and a failure reason otherwise. A URL with
   * a fragment also fails when the document lacks the anchor target. An external URL
   * fails only on proof that it is dead, such as a 404; an outage or bot wall on its
   * host is a warning. External hosts get caching, retries, and per-host serialization;
   * the checks' own server is fetched directly.
   */
  fetchOk: (url: string) => Promise<string | null>;
  /** Reports a problem that does not fail the run, at its source when known. */
  warn: (message: string, at: Source | null) => void;
}

export interface Check {
  name: string;
  /** Called once per crawled page. Failures carry their own source location. */
  page?: (page: Page) => string[];
  /**
   * Called after the crawl, for cross-page validation and network fetches. Failures are
   * streamed through report as they are found; progress reports how many of the check's
   * items have been resolved so far.
   */
  finish?: (
    ctx: Context,
    report: (message: string) => void,
    progress: (done: number, total: number) => void,
  ) => Promise<void>;
}
