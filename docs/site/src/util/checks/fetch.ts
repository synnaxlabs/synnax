// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Context } from "./check.ts";

// Some hosts reject non-browser user agents outright.
const HEADERS = {
  "user-agent":
    "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 " +
    "(KHTML, like Gecko) Chrome/126.0.0.0 Safari/537.36",
  accept: "*/*",
};
const ATTEMPTS = 2;
const TIMEOUT_MS = 5000;

const request = async (url: string, method: string): Promise<number | string> => {
  try {
    const res = await fetch(url, {
      method,
      headers: HEADERS,
      redirect: "follow",
      signal: AbortSignal.timeout(TIMEOUT_MS),
    });
    await res.body?.cancel();
    return res.status;
  } catch (e) {
    return e instanceof Error ? e.message : String(e);
  }
};

const describe = (last: number | string): string =>
  typeof last === "number" ? `HTTP ${last}` : last;

interface Probe {
  reason: string;
  // A connection-level failure (timeout, refused) rather than an HTTP status; these
  // are the slow failures the per-host circuit breaker counts.
  hung: boolean;
}

// A null body is a page that answered but cannot be verified (a 429): the link
// passes and its fragment goes unchecked.
interface Body {
  body: string | null;
}

// Fragment checks need the document itself, so GET with the same retry policy as
// probe.
const fetchBody = async (url: string): Promise<Body | Probe> => {
  let last: number | string = 0;
  for (let attempt = 1; attempt <= ATTEMPTS; attempt++) {
    try {
      const res = await fetch(url, {
        headers: HEADERS,
        redirect: "follow",
        signal: AbortSignal.timeout(TIMEOUT_MS),
      });
      if (res.status < 400) return { body: await res.text() };
      await res.body?.cancel();
      if (res.status === 429) return { body: null };
      last = res.status;
      if (last === 404 || last === 410) break;
    } catch (e) {
      last = e instanceof Error ? e.message : String(e);
    }
    if (attempt < ATTEMPTS)
      await new Promise((resolve) => setTimeout(resolve, 1000 * attempt));
  }
  return { reason: `${url}: ${describe(last)}`, hung: typeof last !== "number" };
};

// Hosts whose pages render anchors client-side; their fragments go unchecked and
// the link is probed for liveness alone.
const SCRIPTED_ANCHOR_HOSTS = ["github.com", "www.github.com"];

const escapeRegExp = (s: string): string => s.replace(/[.*+?^${}()|[\]\\]/g, "\\$&");

// GitHub-style renderers prefix heading ids with user-content- and resolve the bare
// fragment in script, so the prefixed id counts as the anchor.
const hasAnchor = (body: string, fragment: string): boolean =>
  new RegExp(
    `\\s(?:id|name)=["']?(?:user-content-)?${escapeRegExp(fragment)}["'\\s>]`,
  ).test(body);

// Retries transient failures with backoff. A GET follows every failed HEAD: a status
// means the host dislikes the URL, and a dropped connection means it refuses HEAD
// itself, which some hosts do at the TLS layer.
const probe = async (url: string): Promise<Probe | null> => {
  // LinkedIn's bot wall answers 999 for live and dead pages alike, so its links
  // cannot be verified either way. A 429 on any host proves nothing about the link.
  const linkedin = new URL(url).host.endsWith("linkedin.com");
  const ok = (status: number): boolean =>
    status < 400 || status === 429 || (linkedin && status === 999);
  let last: number | string = 0;
  for (let attempt = 1; attempt <= ATTEMPTS; attempt++) {
    last = await request(url, "HEAD");
    if (typeof last !== "number" || !ok(last)) {
      last = await request(url, "GET");
      if (typeof last === "number" && ok(last)) return null;
      if (last === 404 || last === 410) break;
    } else return null;
    if (attempt < ATTEMPTS)
      await new Promise((resolve) => setTimeout(resolve, 1000 * attempt));
  }
  return { reason: `${url}: ${describe(last)}`, hung: typeof last !== "number" };
};

// The checks' own server answers fast and never rate-limits; one plain GET, none of
// the external-host policy.
const probeLocal = async (url: string): Promise<string | null> => {
  const status = await request(url, "GET");
  if (typeof status === "number" && status < 400) return null;
  return `${url}: ${describe(status)}`;
};

// Consecutive connection-level failures on one host before its remaining URLs fail
// immediately instead of burning timeouts.
const BREAKER_LIMIT = 3;

// Concurrent requests allowed per external host: parallel enough that link-heavy
// hosts don't dominate the run, small enough to stay polite.
const HOST_WINDOW = 3;

interface Gate {
  active: number;
  waiting: (() => void)[];
}

// A waiter woken by leave inherits the slot, so active stays exact.
const enter = async (gate: Gate): Promise<void> => {
  if (gate.active < HOST_WINDOW) {
    gate.active += 1;
    return;
  }
  await new Promise<void>((resolve) => gate.waiting.push(resolve));
};

const leave = (gate: Gate): void => {
  const next = gate.waiting.shift();
  if (next != null) next();
  else gate.active -= 1;
};

// Deduplicates by URL and windows requests per host to stay polite with external
// sites; the checks' own server bypasses that policy. A URL with a fragment is
// fetched in full and its anchor target verified against the document, with the
// page shared across fragments of the same URL.
export const createFetcher = (baseURL: string): Context["fetchOk"] => {
  const cache = new Map<string, Promise<string | null>>();
  const bodies = new Map<string, Promise<Body | Probe>>();
  const gates = new Map<string, Gate>();
  const hungCounts = new Map<string, number>();
  return (url) => {
    const cached = cache.get(url);
    if (cached != null) return cached;
    if (url.startsWith(`${baseURL}/`)) {
      const local = probeLocal(url);
      cache.set(url, local);
      return local;
    }
    const parsed = new URL(url);
    const fragment = decodeURIComponent(parsed.hash).replace(/^#/, "");
    parsed.hash = "";
    const page = parsed.toString();
    const host = parsed.host;
    let gate = gates.get(host);
    if (gate == null) gates.set(host, (gate = { active: 0, waiting: [] }));
    const run = async (): Promise<string | null> => {
      await enter(gate);
      try {
        // Checked after enter so queued requests see a breaker tripped mid-flight.
        if ((hungCounts.get(host) ?? 0) >= BREAKER_LIMIT)
          return `${url}: skipped, ${host} stopped answering`;
        // A ":~:" fragment is a text directive, not an element anchor, and hosts
        // that build their anchors in script cannot be checked from static HTML.
        if (
          fragment === "" ||
          fragment.startsWith(":~:") ||
          SCRIPTED_ANCHOR_HOSTS.includes(host)
        ) {
          const res = await probe(url);
          if (res == null) hungCounts.set(host, 0);
          else if (res.hung) hungCounts.set(host, (hungCounts.get(host) ?? 0) + 1);
          return res?.reason ?? null;
        }
        let body = bodies.get(page);
        if (body == null) {
          // Breaker strikes land here, once per page, not once per awaiter.
          body = fetchBody(page).then((res) => {
            if ("reason" in res) {
              if (res.hung) hungCounts.set(host, (hungCounts.get(host) ?? 0) + 1);
            } else hungCounts.set(host, 0);
            return res;
          });
          bodies.set(page, body);
        }
        const res = await body;
        if ("reason" in res) return res.reason;
        if (res.body != null && !hasAnchor(res.body, fragment))
          return `${url}: missing anchor #${fragment}`;
        return null;
      } finally {
        leave(gate);
      }
    };
    const result = run();
    cache.set(url, result);
    return result;
  };
};
