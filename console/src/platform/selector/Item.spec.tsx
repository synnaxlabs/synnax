// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Icon, type Status } from "@synnaxlabs/pluto";
import { fireEvent, screen, waitFor } from "@testing-library/react";
import { type ReactElement } from "react";
import { describe, expect, it, vi } from "vitest";

import { Layout } from "@/platform/layout";
import { type Modals } from "@/platform/modals";
import { Selector } from "@/platform/selector";
import { Session } from "@/session";
import { renderWithConsole } from "@/testutil";

const rename: Modals.PromptRename = async () => null;
const handleError: Status.ErrorHandler = () => {};

const layout: Session.Layout.BaseState = {
  type: "cat",
  name: "Cat",
  location: "mosaic",
};

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

  it("places the layout into the slice when wired to the real placer", async () => {
    const C = Selector.createSimpleItem({ title: "Cat", icon: <Icon.Add />, layout });
    const Host = (): ReactElement => {
      const place = Layout.usePlacer();
      return (
        <C layoutKey="lk" onPlace={place} rename={rename} handleError={handleError} />
      );
    };
    const { store } = await renderWithConsole(<Host />);
    fireEvent.click(screen.getByText("Cat"));
    await waitFor(() => {
      const placed = Session.Layout.select(store.getState(), "lk");
      expect(placed?.type).toBe("cat");
      expect(placed?.name).toBe("Cat");
      expect(placed?.location).toBe("mosaic");
    });
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
