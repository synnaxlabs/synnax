// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import type { status } from "@synnaxlabs/client";
import { Modal } from "@synnaxlabs/lyra/modal";
import { fireEvent, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import { Device } from "@/platform/device";
import { Modals } from "@/platform/modals";
import { findButton, pressEscape, renderModalOpener } from "@/platform/modals/testutil";

interface FooterParams {
  variant: status.Variant;
  onBack: () => void;
}

const Footer: Modals.Content<FooterParams, void> = ({ variant, onBack }) => (
  <Modal.Frame>
    <Device.ConnectFooter
      reason={null}
      variant={variant}
      onConnect={() => {}}
      onBack={onBack}
    />
  </Modal.Frame>
);

const useOpen = Modals.create(Footer);

describe("ConnectFooter", () => {
  it("should hold the dialog and Back while it connects", async () => {
    const onBack = vi.fn();
    await renderModalOpener(useOpen, [{ variant: "loading", onBack }]);
    fireEvent.click(findButton("Back"));
    pressEscape();
    expect(onBack).not.toHaveBeenCalled();
    expect(screen.queryByText("Back")).not.toBeNull();
  });

  it("should let Escape close the dialog when idle", async () => {
    await renderModalOpener(useOpen, [{ variant: "success", onBack: vi.fn() }]);
    pressEscape();
    expect(screen.queryByText("Back")).toBeNull();
  });
});
