import { fireEvent, render } from "@testing-library/react";
import { beforeAll, describe, expect, it, vi } from "vitest";

import { mockBoundingClientRect } from "../../testutil/mocks";

import { List } from ".";

const cols = [
  {
    key: "name",
    label: "Name",
    visible: true,
  },
  {
    key: "age",
    label: "Age",
    visible: true,
  },
];
const data = [
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

const colList = (
  <List data={data}>
    <List.Column.Header columns={cols} />
    <List.Core.Virtual itemHeight={30}>
      {(props) => <List.Column.Item {...props} />}
    </List.Core.Virtual>
  </List>
);

describe("List", () => {
  beforeAll(() => {
    vi.mock("../../util/canvas.ts", () => ({
      getTextWidth: () => 0,
    }));
    Element.prototype.getBoundingClientRect = mockBoundingClientRect(0, 0, 100, 100);
  });
  describe("Column", () => {
    it("should render a column list with the provided items", async () => {
      const c = render(colList);
      expect(c.getByText("Name")).toBeTruthy();
      expect(c.getByText("Age")).toBeTruthy();
      expect(c.getByText("John")).toBeTruthy();
      expect(await c.findByText("Jane")).toBeTruthy();
    });
    it("should allow a user to select an item in the list", async () => {
      const c = render(colList);
      fireEvent.click(c.getByText("John"));
      const selected = await c.findByText("John");
      expect(selected.parentElement?.className).toContain(
        "pluto-list-col-item__container--selected"
      );
    });
    it("should allow a user to deselect an item in the list", async () => {
      const c = render(colList);
      fireEvent.click(c.getByText("John"));
      fireEvent.click(c.getByText("John"));
      const selected = await c.findByText("John");
      expect(selected.parentElement?.className).not.toContain(
        "pluto-list-col-item__container--selected"
      );
    });
    it("should allow a user to select multiple items in the list when holding shift", async () => {
      const c = render(colList);
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
    it("should allow a user to deselect multiple items in the list when holding shift", () => {
      const c = render(colList);
      fireEvent.keyDown(document.body, { key: "Shift" });
      fireEvent.click(c.getByText("John"));
      fireEvent.click(c.getByText("Jack"));
      fireEvent.keyUp(document.body, { key: "Shift" });
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
      const c = render(colList);
      fireEvent.click(c.getByText("Name"));
      const jack = c.getByText("Jack");
      expect(jack.parentElement?.nextSibling?.textContent).toBe("Jane20");
    });
  });
});
