import { fireEvent, render } from "@testing-library/react";
import { expect, describe, it, beforeAll, vi } from "vitest";
import { Select } from ".";
import { mockBoundingClientRect } from "../../testutil/mocks";

const mockColumns = [
  {
    key: "name" as "name",
    label: "Name",
    visible: true,
  },
  {
    key: "age" as "age",
    label: "Age",
    visible: true,
  },
];

const mockOptions = [
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

const selectMultiple = (
  <Select.Multiple columns={mockColumns} options={mockOptions} tagKey="name" />
);

describe("Select", () => {
  beforeAll(() => {
    vi.mock("../../util/canvas.ts", () => ({
      getTextWidth: () => 0,
    }));
    Element.prototype.getBoundingClientRect = mockBoundingClientRect(
      0,
      0,
      100,
      100
    );
  });
  describe("Select.Multiple", () => {
    it("should render a search input", () => {
      const c = render(selectMultiple);
      expect(c.getByPlaceholderText("Search")).toBeTruthy();
    });
    it("should render a list of options when the input area is selected", () => {
      const c = render(selectMultiple);
      fireEvent.click(c.getByPlaceholderText("Search"));
      expect(c.getByText("John")).toBeTruthy();
    });
    it("should not render a list of options when the input area is not selected", () => {
      const c = render(selectMultiple);
      const el = c.getByText("John");
      expect(
        el.parentElement?.parentElement?.parentElement?.parentElement?.className
      ).toContain("hidden");
    });
    it("should allow the user to select an item", async () => {
      const c = render(selectMultiple);
      fireEvent.click(c.getByPlaceholderText("Search"));
      fireEvent.click(c.getByText("John"));
      const j = await c.queryAllByText("John");
      expect(j.length).toBe(2);
    });
    it("should allow the user to remove a selected item", async () => {
      const c = render(selectMultiple);
      fireEvent.click(c.getByPlaceholderText("Search"));
      fireEvent.click(c.getByText("John"));
      const j = await c.findAllByText("John");
      fireEvent.click(j[0].nextSibling as HTMLElement);
      const j2 = await c.queryAllByText("John");
      expect(j2.length).toBe(1);
    });
    it("should allow the user to clear all selections", async () => {
      const c = render(selectMultiple);
      fireEvent.click(c.getByPlaceholderText("Search"));
      fireEvent.click(c.getByText("John"));
      fireEvent.click(c.getByText("James"));
      fireEvent.click(c.getByText("Javier"));
      fireEvent.click(c.getByLabelText("clear"));
      const j = await c.queryAllByText("John");
      const j2 = await c.queryAllByText("James");
      const j3 = await c.queryAllByText("Javier");
      expect(j.length).toBe(1);
      expect(j2.length).toBe(1);
      expect(j3.length).toBe(1);
    });
  });
});
