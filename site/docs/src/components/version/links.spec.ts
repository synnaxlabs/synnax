// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { beforeAll, describe, expect, it } from "vitest";

import { track } from "@/components/version/links";

const ORIGIN = "https://next.docs.synnaxlabs.com";

const link = (): HTMLAnchorElement => {
  const a = document.createElement("a");
  a.className = "tracked";
  a.dataset.origin = ORIGIN;
  a.href = `${ORIGIN}/`;
  a.addEventListener("click", (event) => event.preventDefault());
  document.body.append(a);
  return a;
};

describe("track", () => {
  beforeAll(() => track(".tracked"));

  it("should point a link at the current page on each page load", () => {
    const a = link();
    history.replaceState(null, "", "/reference/control/authority?client=arc#set");
    document.dispatchEvent(new Event("astro:page-load"));
    expect(a.href).toEqual(`${ORIGIN}/reference/control/authority?client=arc#set`);
  });

  it("should follow a query that changed after the page loaded", () => {
    const a = link();
    history.replaceState(null, "", "/reference/");
    document.dispatchEvent(new Event("astro:page-load"));
    history.replaceState(null, "", "/reference/?platform=windows#install");
    a.click();
    expect(a.href).toEqual(`${ORIGIN}/reference/?platform=windows#install`);
  });

  it("should leave other links alone", () => {
    const other = document.createElement("a");
    other.href = "https://example.com/";
    document.body.append(other);
    document.dispatchEvent(new Event("astro:page-load"));
    expect(other.href).toEqual("https://example.com/");
  });
});
