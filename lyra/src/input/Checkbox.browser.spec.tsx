// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ReactElement } from "react";
import { describe, expect, it, vi } from "vitest";
import { userEvent } from "vitest/browser";

import { CSS } from "@/css";
import { Input } from "@/input";
import { renderThemed } from "@/styletest";

describe("Input.Checkbox", () => {
  it.each<[string, (checkbox: ReactElement) => ReactElement]>([
    ["alone", (checkbox) => checkbox],
    ["in a field", (checkbox) => <Input.Item label="Enabled">{checkbox}</Input.Item>],
  ])("should change when its input is clicked %s", async (_, wrap) => {
    const onChange = vi.fn();
    const { getByRole } = await renderThemed(
      wrap(<Input.Checkbox value={false} onChange={onChange} />),
    );
    await userEvent.click(getByRole("checkbox"));
    expect(onChange).toHaveBeenCalledExactlyOnceWith(true);
  });

  it("should start at the edge of its label in a plain table cell", async () => {
    const { container, getByText } = await renderThemed(
      <table>
        <tbody>
          <tr>
            <td>
              <Input.Item label="Enabled">
                <Input.Checkbox value={false} onChange={() => {}} />
              </Input.Item>
            </td>
          </tr>
        </tbody>
      </table>,
    );
    const indicator = container.querySelector(
      `.${CSS.BE("input", "checkbox", "indicator")}`,
    );
    if (indicator == null) throw new Error("The field has no checkbox");
    const label = getByText("Enabled").element().getBoundingClientRect();
    expect(indicator.getBoundingClientRect().left).toBeCloseTo(label.left, 0);
  });

  it("should call onClick once per click", async () => {
    const onClick = vi.fn();
    const { getByRole } = await renderThemed(
      <Input.Checkbox value={false} onChange={() => {}} onClick={onClick} />,
    );
    await userEvent.click(getByRole("checkbox"));
    expect(onClick).toHaveBeenCalledOnce();
  });
});
