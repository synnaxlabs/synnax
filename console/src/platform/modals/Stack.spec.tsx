// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Button } from "@synnaxlabs/pluto";
import { fireEvent, screen, waitFor } from "@testing-library/react";
import { type ReactElement } from "react";
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
});
