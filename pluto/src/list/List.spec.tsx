// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { act, render } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { type ReactElement, useState } from "react";
import { beforeAll, describe, expect, it, vi } from "vitest";

import { List } from "@/list";
import { mockBoundingClientRect } from "@/testutil/dom";
import { Triggers } from "@/triggers";

interface SampleRecord {
  key: string;
  name: string;
  age: number;
}

const cols: Array<List.ColumnSpec<string, SampleRecord>> = [
  { key: "name", name: "Name", visible: true },
  { key: "age", name: "Age", visible: true },
];
const data: SampleRecord[] = [
  { key: "1", name: "John", age: 32 },
  { key: "2", name: "Jane", age: 20 },
  { key: "3", name: "Jack", age: 40 },
];

const TestList = (): ReactElement => {
  const [selected, setSelected] = useState<string[]>([]);
  return (
    <Triggers.Provider>
      <List.List data={data}>
        <List.Selector value={selected} onChange={setSelected}>
          <List.Column.Header columns={cols}>
            <List.Core.Virtual<string> itemHeight={30}>
              {({ key, ...rest }) => <List.Column.Item key={key} {...rest} />}
            </List.Core.Virtual>
          </List.Column.Header>
        </List.Selector>
      </List.List>
    </Triggers.Provider>
  );
};

describe("List", () => {
  beforeAll(() => {
    vi.mock("../../util/canvas.ts", () => ({ textWidth: () => 0 }));
    Element.prototype.getBoundingClientRect = mockBoundingClientRect(0, 0, 100, 100);
  });
  describe("Column", () => {
    it("should render a column list with the provided items", async () => {
      const c = render(<TestList />);
      expect(c.getByText("Name")).toBeTruthy();
      expect(c.getByText("Age")).toBeTruthy();
      expect(c.getByText("John")).toBeTruthy();
      expect(await c.findByText("Jane")).toBeTruthy();
    });
    it("should allow a user to select an item in the list", async () => {
      const user = userEvent.setup();
      const c = render(<TestList />);
      await act(async () => {
        await user.click(c.getByText("John"));
      });
      const selected = await c.findByText("John");
      expect(selected.parentElement?.className).toContain("pluto--selected");
    });
    it("should allow a user to deselect an item in the list", async () => {
      const user = userEvent.setup();
      const c = render(<TestList />);
      await act(async () => {
        await user.click(c.getByText("John"));
        await user.click(c.getByText("John"));
      });
      const selected = await c.findByText("John");
      expect(selected.parentElement?.className).not.toContain("pluto--selected");
    });
    it("should allow a user to select multiple items in the list when holding shift", async () => {
      const user = userEvent.setup();
      const c = render(<TestList />);
      await act(async () => {
        await user.keyboard("[ShiftLeft>]");
        await user.click(c.getByText("John"));
        await user.click(c.getByText("Jack"));
      });
      const selected = await c.findAllByText(/(John|Jack|Jane)/);
      selected.forEach((s) =>
        expect(s.parentElement?.className).toContain("pluto--selected"),
      );
    });
    it("should allow a user to deselect multiple items in the list when holding shift", async () => {
      const user = userEvent.setup();
      const c = render(<TestList />);
      await act(async () => {
        await user.keyboard("[ShiftLeft>]");
        await user.click(c.getByText("John"));
        await user.click(c.getByText("Jack"));
        await user.keyboard("[ShiftLeft>]");
        await new Promise((resolve) => setTimeout(resolve, 450));
        await user.keyboard("[ShiftLeft>]");
        await user.click(c.getByText("John"));
        await user.click(c.getByText("Jack"));
        await user.keyboard("[ShiftLeft>]");
      });
      const selected = c.queryAllByText(/(John|Jack|Jane)/);
      expect(selected.length).toBe(3);
      selected.forEach((s) =>
        expect(s.parentElement?.className).not.toContain("pluto--selected"),
      );
    });
    it("should allow a user to sort the column by name", async () => {
      const user = userEvent.setup();
      const c = render(<TestList />);
      await act(async () => {
        await user.click(c.getByText("Name"));
      });
      const jack = c.getByText("Jack");
      expect(jack.parentElement?.nextSibling?.textContent).toBe("Jane20");
    });
  });
});
