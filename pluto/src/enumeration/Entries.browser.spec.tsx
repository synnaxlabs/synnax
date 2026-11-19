// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { renderThemed } from "@synnaxlabs/lyra/styletest";
import { screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { Enumeration } from "@/enumeration";
import { createSynnaxWrapper } from "@/testutil/Synnax";

/** @returns whether a pointer at the element's center lands on it. */
const takesPointer = (el: Element): boolean => {
  const { left, top, width, height } = el.getBoundingClientRect();
  return el.contains(document.elementFromPoint(left + width / 2, top + height / 2));
};

describe("Enumeration.Table", () => {
  it("should take a pointer on the swatch and the warning of a color cell", async () => {
    const Wrapper = createSynnaxWrapper({ client: null });
    await renderThemed(
      <Wrapper>
        <Enumeration.Table
          value={[{ key: "off", name: "", value: 0, abnormal: false }]}
          onSet={() => {}}
          onAdd={() => {}}
          onRemove={() => {}}
          onMove={() => {}}
        />
      </Wrapper>,
    );
    expect(takesPointer(screen.getByRole("button", { name: "Pick color 1" }))).toBe(
      true,
    );
    expect(takesPointer(screen.getByRole("img", { name: /It has no name/ }))).toBe(
      true,
    );
  });
});
