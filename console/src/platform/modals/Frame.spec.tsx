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
import { describe, expect, it } from "vitest";

import { Modals } from "@/platform/modals";
import { Wrapper } from "@/platform/modals/testutil";

describe("Frame", () => {
  it("should render children inside the modal element and forward className", () => {
    const { baseElement } = render(
      <Dialog.Frame variant="modal" visible>
        <Modals.Frame className="extra">
          <span>frame body</span>
        </Modals.Frame>
      </Dialog.Frame>,
      { wrapper: Wrapper },
    );
    const el = baseElement.querySelector(".console-modal");
    expect(el).not.toBeNull();
    expect(el?.className).toContain("extra");
    expect(screen.getByText("frame body")).toBeTruthy();
  });
});
