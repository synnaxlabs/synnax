// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Icon, type Status } from "@synnaxlabs/pluto";
import { fireEvent, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import { type Modals } from "@/platform/modals";
import { Selector } from "@/platform/selector";
import { type Session } from "@/session";
import { renderWithConsole } from "@/testutil";

const noop = () => {};
const rename = noop as unknown as Modals.PromptRename;
const handleError = noop as unknown as Status.ErrorHandler;

const layout: Session.Layout.BaseState = {
  type: "cat",
  name: "Cat",
  location: "mosaic",
};

describe("Selector.Item", () => {
  it("renders the title and fires onClick", async () => {
    const onClick = vi.fn();
    await renderWithConsole(
      <Selector.Item title="Add Cat" icon={<Icon.Add />} onClick={onClick} />,
    );
    fireEvent.click(screen.getByText("Add Cat"));
    expect(onClick).toHaveBeenCalledTimes(1);
  });
});

describe("Selector.createSimpleItem", () => {
  it("exposes the layout type and visibility hook as statics", () => {
    const useVisible = () => true;
    const C = Selector.createSimpleItem({
      title: "Cat",
      icon: <Icon.Add />,
      layout,
      useVisible,
    });
    expect(C.type).toBe("cat");
    expect(C.useVisible).toBe(useVisible);
  });

  it("places the layout under the given key when clicked", async () => {
    const onPlace = vi.fn();
    const C = Selector.createSimpleItem({ title: "Cat", icon: <Icon.Add />, layout });
    await renderWithConsole(
      <C layoutKey="lk" onPlace={onPlace} rename={rename} handleError={handleError} />,
    );
    fireEvent.click(screen.getByText("Cat"));
    expect(onPlace).toHaveBeenCalledWith({ ...layout, key: "lk" });
  });

  it("renders nothing when useVisible returns false", async () => {
    const C = Selector.createSimpleItem({
      title: "Hidden",
      icon: <Icon.Add />,
      layout,
      useVisible: () => false,
    });
    await renderWithConsole(
      <C layoutKey="lk" onPlace={vi.fn()} rename={rename} handleError={handleError} />,
    );
    expect(screen.queryByText("Hidden")).toBeNull();
  });
});
