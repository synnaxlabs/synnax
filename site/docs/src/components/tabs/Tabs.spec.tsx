// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ReactElement } from "react";
import { renderToString } from "react-dom/server";
import { describe, expect, it } from "vitest";

import { Tabs } from "@/components/tabs/Tabs";

const TABS = [
  { tabKey: "python", name: "Python" },
  { tabKey: "typescript", name: "TypeScript" },
  { tabKey: "cpp", name: "C++" },
];

const keys = (page: ReactElement): (string | undefined)[] => {
  document.body.innerHTML = renderToString(page);
  return [...document.querySelectorAll<HTMLElement>('[role="tab"]')].map(
    (tab) => tab.dataset.tabKey,
  );
};

describe("Tabs", () => {
  it("should show every tab in order by default", () => {
    expect(keys(<Tabs tabs={TABS} />)).toEqual(["python", "typescript", "cpp"]);
  });

  it("should leave out excluded tabs", () => {
    expect(keys(<Tabs tabs={TABS} exclude={["typescript"]} />)).toEqual([
      "python",
      "cpp",
    ]);
  });

  it("should show priority tabs first and keep the order of the rest", () => {
    expect(keys(<Tabs tabs={TABS} priority={["cpp"]} />)).toEqual([
      "cpp",
      "python",
      "typescript",
    ]);
  });
});
