// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import {
  MetricTable,
  type MetricTableColumn,
  type MetricTableData,
} from "@/perf/components/MetricTable";
import { DISPLAY_LIMIT } from "@/perf/constants";

interface TestItem {
  id: string;
  name: string;
  value: number;
}

const TEST_COLUMNS: MetricTableColumn<TestItem>[] = [
  { getValue: (item) => item.name, color: 7 },
  { getValue: (item) => item.value },
];

const createTestData = (count: number): MetricTableData<TestItem> => ({
  data: Array.from({ length: count }, (_, i) => ({
    id: `item-${i}`,
    name: `Item ${i}`,
    value: i * 10,
  })),
  total: count,
  truncated: count > DISPLAY_LIMIT,
});

describe("MetricTable", () => {
  it("should render all rows when data length is under limit", () => {
    const data = createTestData(5);
    const { container } = render(
      <MetricTable result={data} columns={TEST_COLUMNS} getKey={(item) => item.id} />,
    );

    const rows = container.querySelectorAll("tbody tr");
    expect(rows).toHaveLength(5);
  });

  it("should truncate rows when data exceeds display limit", () => {
    const data = createTestData(30);
    const { container } = render(
      <MetricTable result={data} columns={TEST_COLUMNS} getKey={(item) => item.id} />,
    );

    const rows = container.querySelectorAll("tbody tr");
    expect(rows).toHaveLength(DISPLAY_LIMIT);
  });

  it("should show truncation message when data is truncated", () => {
    const data = createTestData(30);
    render(
      <MetricTable result={data} columns={TEST_COLUMNS} getKey={(item) => item.id} />,
    );

    const message = screen.getByText(`Showing ${DISPLAY_LIMIT} of 30`);
    expect(message).toBeDefined();
  });

  it("should not show truncation message when data is under limit", () => {
    const data = createTestData(5);
    render(
      <MetricTable result={data} columns={TEST_COLUMNS} getKey={(item) => item.id} />,
    );

    expect(screen.queryByText(/Showing/)).toBeNull();
  });

  it("should render column values correctly", () => {
    const data = createTestData(2);
    render(
      <MetricTable result={data} columns={TEST_COLUMNS} getKey={(item) => item.id} />,
    );

    expect(screen.getByText("Item 0")).toBeDefined();
    expect(screen.getByText("0")).toBeDefined();
    expect(screen.getByText("Item 1")).toBeDefined();
    expect(screen.getByText("10")).toBeDefined();
  });

  it("should apply tooltips when provided", () => {
    const data = createTestData(1);
    const { container } = render(
      <MetricTable
        result={data}
        columns={TEST_COLUMNS}
        getKey={(item) => item.id}
        getTooltip={(item) => `Tooltip for ${item.name}`}
      />,
    );

    const row = container.querySelector("tbody tr");
    expect(row?.getAttribute("title")).toBe("Tooltip for Item 0");
  });

  it("should render correct number of columns", () => {
    const data = createTestData(1);
    const { container } = render(
      <MetricTable result={data} columns={TEST_COLUMNS} getKey={(item) => item.id} />,
    );

    const cells = container.querySelectorAll("tbody tr:first-child td");
    expect(cells).toHaveLength(2);
  });

  it("should handle empty data", () => {
    const data: MetricTableData<TestItem> = {
      data: [],
      total: 0,
      truncated: false,
    };
    const { container } = render(
      <MetricTable result={data} columns={TEST_COLUMNS} getKey={(item) => item.id} />,
    );

    const rows = container.querySelectorAll("tbody tr");
    expect(rows).toHaveLength(0);
  });
});
