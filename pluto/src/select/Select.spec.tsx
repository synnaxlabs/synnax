// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { fireEvent, render } from "@testing-library/react";
import { type ReactElement, useState } from "react";
import { beforeAll, describe, expect, it, vi } from "vitest";

import { type List } from "@/list";
import { type UseSelectOnChangeExtra } from "@/list/useSelect";
import { Select } from "@/select";
import { DEFAULT_PLACEHOLDER } from "@/select/Single";
import { mockBoundingClientRect } from "@/testutil/dom";
import { Triggers } from "@/triggers";

interface MockRecord {
  key: string;
  name: string;
  age: number;
}

const mockColumns: Array<List.ColumnSpec<string, MockRecord>> = [
  { key: "name", name: "Name", visible: true },
  { key: "age", name: "Age", visible: true },
];

const mockOptions: MockRecord[] = [
  { key: "1", name: "John", age: 32 },
  { key: "2", name: "James", age: 20 },
  { key: "3", name: "Javier", age: 40 },
];

interface SelectMultipleProps
  extends Partial<Select.MultipleProps<string, MockRecord>> {}

const SelectMultiple = (props: SelectMultipleProps): ReactElement => {
  const [value, setValue] = useState<string[]>([]);

  const handleChange = (
    v: string[],
    extra: UseSelectOnChangeExtra<string, MockRecord>,
  ): void => {
    props.onChange?.(v, extra);
    setValue(v);
  };

  return (
    <Triggers.Provider>
      <Select.Multiple<string, MockRecord>
        columns={mockColumns}
        data={mockOptions}
        entryRenderKey="name"
        value={value}
        onChange={handleChange}
        {...props}
      />
    </Triggers.Provider>
  );
};

export interface SelectSingleProps
  extends Partial<Select.SingleProps<string, MockRecord>> {}

const SelectSingle = ({
  onChange,
  value: propsValue,
  ...props
}: SelectSingleProps): ReactElement => {
  const [value, setValue] = useState<string | null>(null);

  const handleChange = (
    v: string,
    extra: UseSelectOnChangeExtra<string, MockRecord>,
  ): void => {
    onChange?.(v, extra);
    setValue(v);
  };

  return (
    <Triggers.Provider>
      <Select.Single<string, MockRecord>
        columns={mockColumns}
        data={mockOptions}
        entryRenderKey="name"
        value={propsValue ?? value}
        onChange={handleChange}
        {...props}
      />
    </Triggers.Provider>
  );
};

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
      expect(c.getByPlaceholderText(DEFAULT_PLACEHOLDER)).toBeTruthy();
    });
    it("should render a list of options when the input area is selected", () => {
      const c = render(<SelectMultiple />);
      fireEvent.click(c.getByPlaceholderText(DEFAULT_PLACEHOLDER));
      expect(c.getByText("John")).toBeTruthy();
    });
    it("should not render a list of options when the input area is not selected", () => {
      const c = render(<SelectMultiple />);
      expect(c.queryByText("John")).toBeFalsy();
    });
    it("should allow the user to select an item", async () => {
      const c = render(<SelectMultiple />);
      fireEvent.click(c.getByPlaceholderText(DEFAULT_PLACEHOLDER));
      fireEvent.click(c.getByText("John"));
      const j = c.queryAllByText("John");
      expect(j.length).toBe(2);
    });
    it("should allow the user to remove a selected item", async () => {
      const c = render(<SelectMultiple />);
      fireEvent.click(c.getByPlaceholderText(DEFAULT_PLACEHOLDER));
      fireEvent.click(c.getByText("John"));
      const j = await c.findAllByText("John");
      fireEvent.click(j[1]);
      const j2 = c.queryAllByText("John");
      expect(j2.length).toBe(1);
    });
    it("should allow the user to clear all selections", async () => {
      const c = render(<SelectMultiple />);
      fireEvent.click(c.getByPlaceholderText(DEFAULT_PLACEHOLDER));
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
    it("should call the onChange handler when the user selects an item", async () => {
      const onChange = vi.fn();
      const c = render(<SelectMultiple onChange={onChange} />);
      fireEvent.click(c.getByPlaceholderText(DEFAULT_PLACEHOLDER));
      fireEvent.click(c.getByText("John"));
      expect(onChange).toHaveBeenCalledWith(["1"], {
        clicked: mockOptions[0].key,
        clickedIndex: 0,
        entries: [mockOptions[0]],
      });
    });
    it("should call the onChange handler when the clears the selection", async () => {
      const onChange = vi.fn();
      const c = render(<SelectMultiple onChange={onChange} />);
      fireEvent.click(c.getByPlaceholderText(DEFAULT_PLACEHOLDER));
      fireEvent.click(c.getByText("John"));
      fireEvent.click(c.getByLabelText("clear"));
      expect(onChange).toHaveBeenCalledWith([], {
        clicked: null,
        clickedIndex: 0,
        entries: [],
      });
    });
  });
  describe("Select.Single", () => {
    it("should render a search input", () => {
      const c = render(<SelectSingle />);
      expect(c.getByPlaceholderText(DEFAULT_PLACEHOLDER)).toBeTruthy();
    });
    it("should render a list of options when the input area is selected", () => {
      const c = render(<SelectSingle />);
      fireEvent.click(c.getByPlaceholderText(DEFAULT_PLACEHOLDER));
      expect(c.getByText("John")).toBeTruthy();
    });
    it("should not render a list of options when the input area is not selected", () => {
      const c = render(<SelectSingle />);
      expect(c.queryByText("John")).toBeFalsy();
    });
    it("should allow the user to select an item", async () => {
      const c = render(<SelectSingle />);
      fireEvent.click(c.getByPlaceholderText(DEFAULT_PLACEHOLDER));
      fireEvent.click(c.getByText("John"));
      const input = await c.findByDisplayValue("John");
      expect(input).toBeTruthy();
    });
    it("should allow the user to clear the selected item", async () => {
      const c = render(<SelectSingle />);
      fireEvent.click(c.getByPlaceholderText(DEFAULT_PLACEHOLDER));
      fireEvent.click(c.getByText("John"));
      fireEvent.click(c.getByLabelText("clear"));
      const input = c.queryByDisplayValue("John");
      expect(input).toBeFalsy();
    });
    it("should call the onChange handler when the user selects an item", async () => {
      const onChange = vi.fn();
      const c = render(<SelectSingle onChange={onChange} />);
      fireEvent.click(c.getByPlaceholderText(DEFAULT_PLACEHOLDER));
      fireEvent.click(c.getByText("John"));
      expect(onChange).toHaveBeenCalled();
    });
    it("should call the onChange handler when the user clears the input", async () => {
      const onChange = vi.fn();
      const c = render(<SelectSingle onChange={onChange} />);
      fireEvent.click(c.getByPlaceholderText(DEFAULT_PLACEHOLDER));
      fireEvent.click(c.getByText("John"));
      fireEvent.click(c.getByLabelText("clear"));
      expect(onChange).toHaveBeenCalledTimes(2);
      expect(onChange).toHaveBeenCalledWith(null, {
        clicked: null,
        clickedIndex: 0,
        entries: [],
      });
    });
  });
});
