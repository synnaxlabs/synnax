// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/app/App.css";

import { renderThemed } from "@synnaxlabs/lyra/styletest";
import { describe, expect, it } from "vitest";

import { Code } from "@/platform/code";

const textWidth = (el: Element): number => {
  const range = document.createRange();
  range.selectNodeContents(el);
  return range.getBoundingClientRect().width;
};

describe("App", () => {
  it("should draw a Geist Mono pair such as ~= in code as two characters", async () => {
    const { getByLabelText } = await renderThemed(
      <>
        <Code.Snippet label="pair" code={[["~="]]} />
        <Code.Snippet label="plain" code={[["ab"]]} />
      </>,
    );
    await document.fonts.load('1em "Geist Mono"');
    const pair = getByLabelText("pair", { exact: true }).element();
    const plain = getByLabelText("plain", { exact: true }).element();
    expect(textWidth(pair)).toBe(textWidth(plain));
  });
});
