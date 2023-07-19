// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { ReactElement, useState } from "react";

import { fireEvent, render } from "@testing-library/react";
import { beforeAll, describe, expect, it, vi } from "vitest";

import { Select } from ".";

import { ListColumn } from "@/core/std/List";
import { Triggers } from "@/core/triggers";
import { mockBoundingClientRect } from "@/testutil/dom";

interface MockRecord {
  key: string;
  name: string;
  age: number;
}

const mockColumns: Array<ListColumn<string, MockRecord>> = [
  {
    key: "name",
    name: "Name",
    visible: true,
  },
  {
    key: "age",
    name: "Age",
    visible: true,
  },
];

const mockOptions: MockRecord[] = [
  {
    key: "1",
    name: "John",
    age: 32,
  },
  {
    key: "2",
    name: "James",
    age: 20,
  },
  {
    key: "3",
    name: "Javier",
    age: 40,
  },
];

const SelectMultiple = (): ReactElement => {
  const [value, setValue] = useState<readonly string[]>([]);
  return (
    <Triggers.Provider>
      <Select.Multiple<string, MockRecord>
        columns={mockColumns}
        data={mockOptions}
        tagKey="name"
        value={value}
        onChange={setValue}
      />
    </Triggers.Provider>
  );
};

const PLACEHOLDER = "Search...";

describe("Select", () => {
  beforeAll(() => {
    vi.mock("../../util/canvas.ts", () => ({
      textWidth: () => 0,
    }));
    Element.prototype.getBoundingClientRect = mockBoundingClientRect(0, 0, 100, 100);
  });
  describe("Select.Multiple", () => {
    it("should render a search input", () => {
      const c = render(<SelectMultiple />);
      expect(c.getByPlaceholderText(PLACEHOLDER)).toBeTruthy();
    });
    it("should render a list of options when the input area is selected", () => {
      const c = render(<SelectMultiple />);
      fireEvent.click(c.getByPlaceholderText(PLACEHOLDER));
      expect(c.getByText("John")).toBeTruthy();
    });
    it("should not render a list of options when the input area is not selected", () => {
      const c = render(<SelectMultiple />);
      const el = c.getByText("John");
      expect(
        el.parentElement?.parentElement?.parentElement?.parentElement?.className
      ).toContain("hidden");
    });
    it("should allow the user to select an item", async () => {
      const c = render(<SelectMultiple />);
      fireEvent.click(c.getByPlaceholderText(PLACEHOLDER));
      fireEvent.click(c.getByText("John"));
      const j = c.queryAllByText("John");
      expect(j.length).toBe(2);
    });
    it("should allow the user to remove a selected item", async () => {
      const c = render(<SelectMultiple />);
      fireEvent.click(c.getByPlaceholderText(PLACEHOLDER));
      fireEvent.click(c.getByText("John"));
      const j = await c.findAllByText("John");
      fireEvent.click(j[0].nextSibling as HTMLElement);
      const j2 = c.queryAllByText("John");
      expect(j2.length).toBe(1);
    });
    it("should allow the user to clear all selections", async () => {
      const c = render(<SelectMultiple />);
      fireEvent.click(c.getByPlaceholderText(PLACEHOLDER));
      fireEvent.click(c.getByText("John"));
      fireEvent.click(c.getByText("James"));
      fireEvent.click(c.getByText("Javier"));
      fireEvent.click(c.getByLabelText("clear"));
      const j = c.queryAllByText("John");
      const j2 = c.queryAllByText("James");
      const j3 = c.queryAllByText("Javier");
      expect(j.length).toBe(1);
      expect(j2.length).toBe(1);
      expect(j3.length).toBe(1);
    });
  });
});
