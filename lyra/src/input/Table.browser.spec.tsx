// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it, vi } from "vitest";

import { Input } from "@/input";
import { computed, renderThemed } from "@/styletest";

const renderWide = async (widths: (number | undefined)[]): Promise<HTMLTableElement> =>
  (
    await renderThemed(
      <div style={{ width: "2000px" }}>
        <Input.Table value={[widths.map((_, i) => i)]} onChange={vi.fn()}>
          {widths.map((width, i) => (
            <Input.TableColumn key={i} name={`Column ${i}`} width={width} />
          ))}
        </Input.Table>
      </div>,
    )
  ).container.querySelector("table")!;

describe("Input.Table", () => {
  it("should keep declared column widths in a wider container", async () => {
    const table = await renderWide([10, 12]);
    expect(table.getBoundingClientRect().width).toBe(
      parseFloat(computed(table, "min-width")),
    );
  });

  it("should fill a wider container when a column has no width", async () => {
    const table = await renderWide([10, undefined]);
    expect(table.getBoundingClientRect().width).toBe(2000);
  });
});
