// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Dialog } from "@synnaxlabs/pluto";
import { render, screen } from "@testing-library/react";
import { type ReactNode } from "react";
import { describe, expect, it } from "vitest";

import { Frame, type FrameProps } from "@/service/modals/Frame";
import { Wrapper } from "@/service/modals/testutil";

const renderFrame = (props: Partial<FrameProps> = {}, children?: ReactNode) =>
  render(
    <Dialog.Frame variant="modal" visible>
      <Frame {...props}>{children}</Frame>
    </Dialog.Frame>,
    { wrapper: Wrapper },
  );

describe("Frame", () => {
  it("should render its children", () => {
    renderFrame({}, <span>frame body</span>);
    expect(screen.getByText("frame body")).toBeTruthy();
  });

  it("should apply the modal block class", () => {
    const { baseElement } = renderFrame({}, <span>frame body</span>);
    expect(baseElement.querySelector(".console-modal")).not.toBeNull();
  });

  it("should forward a custom className", () => {
    const { baseElement } = renderFrame(
      { className: "extra" },
      <span>frame body</span>,
    );
    const el = baseElement.querySelector(".console-modal");
    expect(el?.className).toContain("extra");
  });
});
