// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { useState } from "react";

import { fireEvent, render } from "@testing-library/react";
import { beforeAll, describe, expect, it, vi } from "vitest";

import { mockBoundingClientRect } from "../../testutil/dom";

import { List, ListColumn } from ".";

import { Triggers } from "@/triggers";

interface SampleRecord {
  key: string;
  name: string;
  age: number;
}

const cols: Array<ListColumn<SampleRecord>> = [
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
const data: SampleRecord[] = [
  {
    key: "1",
    name: "John",
    age: 32,
  },
  {
    key: "2",
    name: "Jane",
    age: 20,
  },
  {
    key: "3",
    name: "Jack",
    age: 40,
  },
];

const TestList = (): JSX.Element => {
  const [selected, setSelected] = useState<readonly string[]>([]);
  return (
    <Triggers.Provider>
      <List data={data}>
        <List.Selector value={selected} onChange={setSelected} />
        <List.Column.Header columns={cols} />
        <List.Core.Virtual itemHeight={30}>
          {(props) => <List.Column.Item {...props} />}
        </List.Core.Virtual>
      </List>
    </Triggers.Provider>
  );
};

describe("List", () => {
  beforeAll(() => {
    vi.mock("../../util/canvas.ts", () => ({
      textWidth: () => 0,
    }));
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
      const c = render(<TestList />);
      fireEvent.click(c.getByText("John"));
      const selected = await c.findByText("John");
      expect(selected.parentElement?.className).toContain(
        "pluto-list-col-item__container--selected"
      );
    });
    it("should allow a user to deselect an item in the list", async () => {
      const c = render(<TestList />);
      fireEvent.click(c.getByText("John"));
      fireEvent.click(c.getByText("John"));
      const selected = await c.findByText("John");
      expect(selected.parentElement?.className).not.toContain(
        "pluto-list-col-item__container--selected"
      );
    });
    it("should allow a user to select multiple items in the list when holding shift", async () => {
      const c = render(<TestList />);
      fireEvent.keyDown(document.body, { key: "Shift" });
      fireEvent.click(c.getByText("John"));
      fireEvent.click(c.getByText("Jack"));
      fireEvent.keyUp(document.body, { key: "Shift" });
      const selected = await c.findAllByText(/(John|Jack|Jane)/);
      expect(selected.length).toBe(3);
      selected.forEach((s) => {
        expect(s.parentElement?.className).toContain(
          "pluto-list-col-item__container--selected"
        );
      });
    });
    it("should allow a user to deselect multiple items in the list when holding shift", async () => {
      const c = render(<TestList />);
      fireEvent.keyDown(document.body, { key: "Shift" });
      fireEvent.click(c.getByText("John"));
      fireEvent.click(c.getByText("Jack"));
      fireEvent.keyUp(document.body, { key: "Shift" });
      await new Promise((resolve) => setTimeout(resolve, 450));
      fireEvent.keyDown(document.body, { key: "Shift" });
      fireEvent.click(c.getByText("John"));
      fireEvent.click(c.getByText("Jack"));
      fireEvent.keyUp(document.body, { key: "Shift" });
      const selected = c.queryAllByText(/(John|Jack|Jane)/);
      expect(selected.length).toBe(3);
      selected.forEach((s) => {
        expect(s.parentElement?.className).not.toContain(
          "pluto-list-col-item__container--selected"
        );
      });
    });
    it("should allow a user to sort the column by name", () => {
      const c = render(<TestList />);
      fireEvent.click(c.getByText("Name"));
      const jack = c.getByText("Jack");
      expect(jack.parentElement?.nextSibling?.textContent).toBe("Jane20");
    });
  });
});
