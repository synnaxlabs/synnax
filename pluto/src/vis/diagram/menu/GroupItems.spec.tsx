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

import { Menu } from "@/menu";
import { GroupItems, type GroupItemsProps } from "@/vis/diagram/menu/GroupItems";

const renderItems = (props: Partial<GroupItemsProps> = {}): ReturnType<typeof render> =>
  render(
    <Menu.Menu>
      <GroupItems
        group={props.group ?? vi.fn()}
        ungroup={props.ungroup ?? vi.fn()}
        canGroup={props.canGroup ?? true}
        canUngroup={props.canUngroup ?? true}
      />
    </Menu.Menu>,
  );

describe("Diagram.Menu.GroupItems", () => {
  it("should render the group and ungroup entries when both apply", () => {
    const { getByText } = renderItems();
    expect(getByText("Group")).toBeDefined();
    expect(getByText("Ungroup")).toBeDefined();
  });

  it("should call group and ungroup when the entries are clicked", () => {
    const group = vi.fn();
    const ungroup = vi.fn();
    const { getByText } = renderItems({ group, ungroup });
    fireEvent.click(getByText("Group"));
    expect(group).toHaveBeenCalledTimes(1);
    fireEvent.click(getByText("Ungroup"));
    expect(ungroup).toHaveBeenCalledTimes(1);
  });

  it("should hide each entry when it does not apply", () => {
    const { queryByText } = renderItems({ canGroup: false, canUngroup: false });
    expect(queryByText("Group")).toBeNull();
    expect(queryByText("Ungroup")).toBeNull();
  });

  it("should show only the entry that applies", () => {
    const { getByText, queryByText } = renderItems({ canUngroup: false });
    expect(getByText("Group")).toBeDefined();
    expect(queryByText("Ungroup")).toBeNull();
  });
});
