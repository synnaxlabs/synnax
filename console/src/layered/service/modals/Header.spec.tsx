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

import { Header } from "@/layered/service/modals/Header";
import { Wrapper } from "@/layered/service/modals/testutil";

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
  it("should render a breadcrumb segment for each dotted name part", () => {
    renderHeader(<Header>Group.Channel.Name</Header>);
    expect(screen.getByText("Group")).toBeTruthy();
    expect(screen.getByText("Channel")).toBeTruthy();
    expect(screen.getByText("Name")).toBeTruthy();
  });

  it("should render the leading icon when one is provided", () => {
    const { baseElement } = renderHeader(
      <Header icon={<Icon.Add className="test-icon" />}>Title</Header>,
    );
    expect(baseElement.querySelector(".test-icon")).not.toBeNull();
  });

  it("should not render an icon segment when no icon is provided", () => {
    const { baseElement } = renderHeader(<Header>Title</Header>);
    expect(baseElement.querySelector(".test-icon")).toBeNull();
  });

  it("should dismiss the dialog when the close button is clicked", () => {
    const { baseElement, onVisibleChange } = renderHeader(<Header>Title</Header>);
    const close = baseElement.querySelector("button");
    expect(close).not.toBeNull();
    fireEvent.click(close as HTMLButtonElement);
    expect(onVisibleChange).toHaveBeenCalled();
  });
});
