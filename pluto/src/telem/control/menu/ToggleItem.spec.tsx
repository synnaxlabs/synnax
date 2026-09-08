// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ReactElement } from "react";
import { assert, describe, expect, it } from "vitest";

import { Control } from "@/telem/control";
import { control } from "@/telem/control/aether";
import { render } from "@/testutil/render";

const CONTROLLER_NAME = "Valve Controller";
const OPTIONS = {
  synnax: { props: null },
  registry: {
    [control.Controller.TYPE]: control.Controller,
    [control.Colors.TYPE]: control.Colors,
  },
};

const renderItem = (
  item: ReactElement,
  props: Partial<Control.ControllerProps> = {},
): HTMLElement => {
  const c = render(
    <Control.Colors>
      <Control.Controller name={CONTROLLER_NAME} {...props}>
        {item}
      </Control.Controller>
    </Control.Colors>,
    OPTIONS,
  );
  const found = c.queryByRole("menuitem");
  assert(found != null, "the toggle item did not render");
  return found;
};

describe("Control.Menu.ToggleItem", () => {
  it("should offer control", () => {
    const item = renderItem(<Control.Menu.ToggleItem />);
    expect(item.textContent).toContain("Take control");
    expect(item.getAttribute("aria-disabled")).toBeNull();
  });

  it("should still offer control while the controller is disabled", () => {
    const item = renderItem(<Control.Menu.ToggleItem />, { disabled: true });
    expect(item.textContent).toContain("Take control");
    expect(item.getAttribute("aria-disabled")).toBeNull();
  });

  it("should bar control when the caller disables it", () => {
    const item = renderItem(<Control.Menu.ToggleItem disabled />);
    expect(item.getAttribute("aria-disabled")).toEqual("true");
  });

  it("should render nothing outside a controller", () => {
    const c = render(<Control.Menu.ToggleItem />, OPTIONS);
    expect(c.queryByRole("menuitem")).toBeNull();
  });
});
