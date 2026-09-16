// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it, vi } from "vitest";

import { type Context } from "./check.ts";
import { notes } from "./notes.ts";

// No page source exists for this route, so failures carry the bare route.
const ROUTE = "/spec/notes";
const CTX = {} as Context;

const note = (body: string, variant = "info"): string =>
  `<div class="pluto-note pluto--${variant} pluto-flex">${body}</div>`;

const run = (html: string): string[] =>
  notes(false).page?.({ route: ROUTE, html }) ?? [];

describe("notes", () => {
  it("should accept a note with block content", () => {
    expect(run(note("<p>Labels are read-only.</p>"))).toEqual([]);
  });

  it("should accept a note with inline content", () => {
    expect(run(note("Labels are read-only."))).toEqual([]);
  });

  it("should flag a note without a known variant", () => {
    const failures = run(note("<p>Hi.</p>", "success"));
    expect(failures).toHaveLength(1);
    expect(failures[0]).toContain(ROUTE);
    expect(failures[0]).toContain("variant is not one of info, warning, error");
  });

  it("should flag an empty note", () => {
    const failures = run(note("  "));
    expect(failures).toEqual([`${ROUTE} - note is empty`]);
  });

  it("should treat a note holding only empty elements as empty", () => {
    expect(run(note("<p></p>"))).toEqual([`${ROUTE} - note is empty`]);
  });

  it("should not read past a nested div when finding the note's end", () => {
    const body = '<p>Intro.</p><div class="astro-code-wrapper"><pre>x</pre></div>';
    expect(run(`${note(body)}${note("", "warning")}`)).toEqual([
      `${ROUTE} - note is empty`,
    ]);
  });

  it("should ignore divs that are not notes", () => {
    expect(run('<div class="pluto-flex">loose text</div>')).toEqual([]);
  });

  it("should report the canary only on a full crawl with no notes", async () => {
    const report = vi.fn();
    await notes(true).finish?.(CTX, report, vi.fn());
    expect(report).toHaveBeenCalledWith("no notes found: markup has changed");
    report.mockClear();
    await notes(false).finish?.(CTX, report, vi.fn());
    expect(report).not.toHaveBeenCalled();
  });

  it("should not report the canary once a note has been seen", async () => {
    const check = notes(true);
    check.page?.({ route: ROUTE, html: note("<p>Seen.</p>") });
    const report = vi.fn();
    await check.finish?.(CTX, report, vi.fn());
    expect(report).not.toHaveBeenCalled();
  });
});
