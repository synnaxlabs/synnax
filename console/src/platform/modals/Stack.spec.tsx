// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Button, Dialog } from "@synnaxlabs/pluto";
import { fireEvent, screen, waitFor } from "@testing-library/react";
import { lazy, type ReactElement } from "react";
import { describe, expect, it } from "vitest";

import { Modals } from "@/platform/modals";
import { renderWithModals } from "@/platform/modals/testutil";

interface OpenerProps {
  text: string;
}

const Content: Modals.Content<OpenerProps, void> = ({ text, close }) => (
  <div>
    <span>{text}</span>
    <Button.Button onClick={() => close()}>close-{text}</Button.Button>
  </div>
);

const useOpen = Modals.create(Content);

const Opener = ({ text }: OpenerProps): ReactElement => {
  const open = useOpen();
  return <Button.Button onClick={() => open({ text })}>open-{text}</Button.Button>;
};

const clickText = (text: string): void => {
  const el = screen.getByText(text).closest("button") ?? screen.getByText(text);
  fireEvent.click(el);
};

describe("Stack", () => {
  it("should render nothing when no modals are open", () => {
    renderWithModals(<Opener text="A" />);
    expect(screen.queryByText("A")).toBeNull();
  });

  it("should mount a modal for each open entry", async () => {
    renderWithModals(
      <>
        <Opener text="A" />
        <Opener text="B" />
      </>,
    );
    clickText("open-A");
    clickText("open-B");
    await waitFor(() => {
      expect(screen.getByText("A")).toBeTruthy();
      expect(screen.getByText("B")).toBeTruthy();
    });
  });

  it("should remove only the dismissed modal from the DOM", async () => {
    renderWithModals(
      <>
        <Opener text="A" />
        <Opener text="B" />
      </>,
    );
    clickText("open-A");
    clickText("open-B");
    await waitFor(() => expect(screen.getByText("A")).toBeTruthy());
    clickText("close-A");
    await waitFor(() => expect(screen.queryByText("A")).toBeNull());
    expect(screen.getByText("B")).toBeTruthy();
  });

  it("should dismiss a modal when its dialog requests close", async () => {
    const DialogCloser: Modals.Content<Record<never, never>, void> = () => {
      const { close } = Dialog.useContext();
      return <Button.Button onClick={() => close()}>dialog-close</Button.Button>;
    };
    const useOpenCloser = Modals.create(DialogCloser);
    const CloserOpener = (): ReactElement => {
      const open = useOpenCloser();
      return <Button.Button onClick={() => open()}>open-closer</Button.Button>;
    };
    renderWithModals(<CloserOpener />);
    clickText("open-closer");
    await waitFor(() => expect(screen.getByText("dialog-close")).toBeTruthy());
    clickText("dialog-close");
    await waitFor(() => expect(screen.queryByText("dialog-close")).toBeNull());
  });

  it("should suspense-render lazily loaded modal content", async () => {
    const Lazy = lazy(async () => ({
      default: (): ReactElement => <span>lazy loaded</span>,
    }));
    const LazyContent: Modals.Content<Record<never, never>, void> = () => <Lazy />;
    const useOpenLazy = Modals.create(LazyContent);
    const LazyOpener = (): ReactElement => {
      const open = useOpenLazy();
      return <Button.Button onClick={() => open()}>open-lazy</Button.Button>;
    };
    renderWithModals(<LazyOpener />);
    clickText("open-lazy");
    await waitFor(() => expect(screen.getByText("lazy loaded")).toBeTruthy());
  });
});
