// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { render } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import { List } from "@/list";

const COLUMNS: List.Column[] = [
  { name: "Type", width: "30rem", field: true },
  { width: "2.5rem" },
  { name: "Channel", width: "1fr" },
];

describe("List.Columns", () => {
  describe("Cells", () => {
    it("should lay out its children on the declared widths", () => {
      const c = render(
        <List.Columns columns={COLUMNS}>
          <List.Cells data-testid="cells" />
        </List.Columns>,
      );
      expect(c.getByTestId("cells").style.gridTemplateColumns).toBe("30rem 2.5rem 1fr");
    });

    it("should throw outside of List.Columns", () => {
      vi.spyOn(console, "error").mockImplementation(() => {});
      expect(() => render(<List.Cells />)).toThrow(
        "List.Cells must be used within List.Columns",
      );
      vi.restoreAllMocks();
    });
  });

  describe("Headings", () => {
    const headings = (checkable?: boolean): HTMLElement => {
      const c = render(
        <List.Columns columns={COLUMNS}>
          <List.Headings checkable={checkable} />
        </List.Columns>,
      );
      return c.container.querySelector<HTMLElement>(".lyra-list__headings")!;
    };

    it("should name each column in order, leaving a nameless column empty", () => {
      const names = [...headings().querySelectorAll(".lyra-list__cells > span")].map(
        (s) => s.textContent,
      );
      expect(names).toEqual(["Type", "", "Channel"]);
    });

    it("should line up the heading of a field column with the field's text", () => {
      const spans = headings().querySelectorAll(".lyra-list__cells > span");
      expect(spans[0].classList.contains("lyra-list__heading--field")).toBe(true);
      expect(spans[2].classList.contains("lyra-list__heading--field")).toBe(false);
    });

    it("should leave room for the checkbox of checkable items", () => {
      expect(headings(true).querySelector(".lyra-list__headings-check")).not.toBeNull();
    });

    it("should leave no room for a checkbox by default", () => {
      expect(headings().querySelector(".lyra-list__headings-check")).toBeNull();
    });
  });
});
