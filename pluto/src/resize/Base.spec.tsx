// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type location } from "@synnaxlabs/x";
import { render } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { Resize } from "@/resize";

const paneOf = (c: ReturnType<typeof render>): HTMLElement => {
  const pane = c.container.querySelector<HTMLElement>(".pluto-resize");
  if (pane == null) throw new Error("resize pane not found");
  return pane;
};

describe("Resize.Base", () => {
  it("should size the width for a horizontal location", () => {
    const c = render(<Resize.Base location="left" size={120} />);
    expect(paneOf(c).style.width).toEqual("120px");
    expect(paneOf(c).style.height).toEqual("");
  });

  it("should size the height for a vertical location", () => {
    const c = render(<Resize.Base location="top" size={120} />);
    expect(paneOf(c).style.height).toEqual("120px");
    expect(paneOf(c).style.width).toEqual("");
  });

  it("should size as a percentage when decimal is set", () => {
    const c = render(<Resize.Base location="left" size={0.5} decimal />);
    expect(paneOf(c).style.width).toEqual("50%");
  });

  it("should follow the location's axis when reused with a flipped direction", () => {
    const renderAt = (loc: location.Crude) => (
      <Resize.Base location={loc} size={0.5} decimal />
    );
    const c = render(renderAt("top"));
    expect(paneOf(c).style.height).toEqual("50%");
    expect(paneOf(c).style.width).toEqual("");
    c.rerender(renderAt("left"));
    expect(paneOf(c).style.width).toEqual("50%");
    expect(paneOf(c).style.height).toEqual("");
  });
});
