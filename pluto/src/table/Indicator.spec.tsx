// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { fireEvent, render } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import { Indicator } from "@/table/Indicator";

const renderIndicator = (props: Partial<React.ComponentProps<typeof Indicator>>) =>
  render(
    <table>
      <tbody>
        <tr>
          <Indicator
            direction="x"
            index={0}
            value={80}
            editable
            onChange={() => {}}
            onSelect={() => {}}
            {...props}
          />
        </tr>
      </tbody>
    </table>,
  );

describe("Indicator", () => {
  it("renders the resize button when editable", () => {
    const { container } = renderIndicator({ editable: true });
    expect(container.querySelector("button")).not.toBeNull();
  });

  it("omits the resize button when not editable", () => {
    const { container } = renderIndicator({ editable: false });
    expect(container.querySelector("button")).toBeNull();
  });

  it("forwards click and context menu to onSelect with the event", () => {
    const onSelect = vi.fn();
    const { container } = renderIndicator({ editable: false, onSelect });
    const td = container.querySelector("td.pluto-table__resizer");
    fireEvent.click(td!);
    fireEvent.contextMenu(td!);
    expect(onSelect).toHaveBeenCalledTimes(2);
    expect(onSelect).toHaveBeenNthCalledWith(
      1,
      0,
      expect.objectContaining({ type: "click" }),
    );
    expect(onSelect).toHaveBeenNthCalledWith(
      2,
      0,
      expect.objectContaining({ type: "contextmenu" }),
    );
  });
});
