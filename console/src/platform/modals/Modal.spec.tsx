// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Button, Dialog } from "@synnaxlabs/pluto";
import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { lazy, type ReactElement } from "react";
import { describe, expect, it, vi } from "vitest";

import { Modal } from "@/platform/modals/Modal";
import { Wrapper } from "@/platform/modals/testutil";
import { type Session } from "@/session";

const entryWith = (
  render: () => ReactElement,
  dismiss = vi.fn(),
): Session.Modals.Entry => ({ key: "test", render, dismiss });

describe("Modal", () => {
  it("should render the entry's content", () => {
    render(<Modal {...entryWith(() => <span>modal content</span>)} />, {
      wrapper: Wrapper,
    });
    expect(screen.getByText("modal content")).toBeTruthy();
  });

  it("should dismiss the entry when the dialog requests close", () => {
    const dismiss = vi.fn();
    const Closer = (): ReactElement => {
      const { close } = Dialog.useContext();
      return <Button.Button onClick={close}>close</Button.Button>;
    };
    render(<Modal {...entryWith(() => <Closer />, dismiss)} />, {
      wrapper: Wrapper,
    });
    fireEvent.click(screen.getByText("close").closest("button") as HTMLButtonElement);
    expect(dismiss).toHaveBeenCalled();
  });

  it("should provide a suspense boundary for suspending content", async () => {
    const Lazy = lazy(async () => ({
      default: (): ReactElement => <span>lazy loaded</span>,
    }));
    render(<Modal {...entryWith(() => <Lazy />)} />, { wrapper: Wrapper });
    await waitFor(() => expect(screen.getByText("lazy loaded")).toBeTruthy());
  });
});
