// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Dialog, Icon } from "@synnaxlabs/pluto";
import { fireEvent, render, screen } from "@testing-library/react";
import { type ReactNode } from "react";
import { describe, expect, it, vi } from "vitest";

import { Modals } from "@/platform/modals";
import { Wrapper } from "@/platform/modals/testutil";

const renderHeader = (ui: ReactNode, onVisibleChange = vi.fn()) => ({
  ...render(
    <Dialog.Frame variant="modal" visible onVisibleChange={onVisibleChange}>
      {ui}
    </Dialog.Frame>,
    { wrapper: Wrapper },
  ),
  onVisibleChange,
});

describe("Header", () => {
  it("should split a dotted name into one breadcrumb segment per part", () => {
    renderHeader(<Modals.Header>Group.Channel.Name</Modals.Header>);
    expect(screen.getByText("Group")).toBeTruthy();
    expect(screen.getByText("Channel")).toBeTruthy();
    expect(screen.getByText("Name")).toBeTruthy();
  });

  it("should render a leading icon segment only when an icon is provided", () => {
    const withIcon = renderHeader(
      <Modals.Header icon={<Icon.Add className="test-icon" />}>Title</Modals.Header>,
    );
    expect(withIcon.baseElement.querySelector(".test-icon")).not.toBeNull();
    withIcon.unmount();
    const withoutIcon = renderHeader(<Modals.Header>Title</Modals.Header>);
    expect(withoutIcon.baseElement.querySelector(".test-icon")).toBeNull();
  });

  it("should dismiss the dialog when the close button is clicked", () => {
    const { baseElement, onVisibleChange } = renderHeader(
      <Modals.Header>Title</Modals.Header>,
    );
    const close = baseElement.querySelector("button");
    expect(close).not.toBeNull();
    fireEvent.click(close as HTMLButtonElement);
    expect(onVisibleChange).toHaveBeenCalledWith(false);
  });
});
