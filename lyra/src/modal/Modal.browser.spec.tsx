// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import type { ReactNode } from "react";
import { describe, expect, it, vi } from "vitest";
import { page } from "vitest/browser";

import { Button } from "@/button";
import { CSS } from "@/css";
import { Dialog } from "@/dialog";
import { Icon } from "@/icon";
import { Modal } from "@/modal";
import { Nav } from "@/nav";
import { computed, renderThemed } from "@/styletest";
import { Tooltip } from "@/tooltip";
import { Triggers } from "@/triggers";

const renderModal = async (
  body: ReactNode = "Body",
  header: ReactNode = <Modal.Header>Title</Modal.Header>,
): Promise<Element> => {
  await renderThemed(
    <Triggers.Provider>
      <Dialog.Frame variant="modal" visible>
        <Modal.Frame>
          {header}
          <Modal.Body>{body}</Modal.Body>
          <Modal.Footer>
            <Nav.Bar.End>
              <Button.Button>Save</Button.Button>
            </Nav.Bar.End>
          </Modal.Footer>
        </Modal.Frame>
      </Dialog.Frame>
    </Triggers.Provider>,
  );
  return page.getByRole("dialog").element();
};

const px = (el: Element, property: string): number =>
  parseFloat(computed(el, property));

const part = (parent: Element, block: string, element: string): Element =>
  parent.querySelector(`.${CSS.BE(block, element)}`)!;

describe("Modal", () => {
  it("should inset the header and footer by one shared amount", async () => {
    const modal = await renderModal();
    const header = part(modal, "modal", "header");
    const footer = part(modal, "modal", "footer");
    const inset = px(header, "padding-top");
    expect(inset).toBeGreaterThan(0);
    expect(px(header, "padding-bottom")).toBe(inset);
    expect(px(part(header, "navbar", "content"), "padding-left")).toBe(inset);
    expect(px(footer, "padding-bottom")).toBe(inset);
    expect(px(part(footer, "navbar", "content"), "padding-right")).toBe(inset);
  });

  it("should fit the window and scroll a tall body", async () => {
    const modal = await renderModal(
      <div style={{ height: 3000, flexShrink: 0 }}>Tall</div>,
    );
    await expect
      .poll(() => modal.getBoundingClientRect().bottom)
      .toBeLessThanOrEqual(window.innerHeight);
    const body = part(modal, "modal", "body");
    expect(body.scrollHeight).toBeGreaterThan(body.clientHeight);
    expect(computed(body, "overflow-y")).toBe("auto");
  });

  it("should keep the close button in the header when the badge is too wide", async () => {
    await page.viewport(400, 600);
    const badge = (
      <span style={{ minWidth: 0, overflow: "hidden", whiteSpace: "nowrap" }}>
        {"location ".repeat(40)}
      </span>
    );
    const modal = await renderModal(
      "Body",
      <Modal.Header badge={badge}>Title</Modal.Header>,
    );
    const header = part(modal, "modal", "header");
    const close = page.getByRole("button", { name: "Close" }).element();
    expect(close.getBoundingClientRect().right).toBeLessThanOrEqual(
      header.getBoundingClientRect().right,
    );
  });
});

describe("Modal", () => {
  const renderModal = (ui: ReactNode, onVisibleChange = vi.fn()) => ({
    ...render(
      <Triggers.Provider>
        <Dialog.Frame variant="modal" visible onVisibleChange={onVisibleChange}>
          {ui}
        </Dialog.Frame>
      </Triggers.Provider>,
    ),
    onVisibleChange,
  });

  describe("Frame", () => {
    it("should render children inside the modal element and forward className", () => {
      const { baseElement } = renderModal(
        <Modal.Frame className="extra">
          <span>frame body</span>
        </Modal.Frame>,
      );
      const el = baseElement.querySelector(".lyra-modal");
      expect(el).not.toBeNull();
      expect(el?.className).toContain("extra");
      expect(screen.getByText("frame body")).toBeTruthy();
    });
  });

  describe("Header", () => {
    it("should split a dotted name into one breadcrumb segment per part", () => {
      renderModal(<Modal.Header>Group.Channel.Name</Modal.Header>);
      expect(screen.getByText("Group")).toBeTruthy();
      expect(screen.getByText("Channel")).toBeTruthy();
      expect(screen.getByText("Name")).toBeTruthy();
    });

    it("should keep pre-split segments whole even when they contain dots", () => {
      renderModal(<Modal.Header>{["Role", "Assign", "user.name"]}</Modal.Header>);
      expect(screen.getByText("user.name")).toBeTruthy();
    });

    it("should draw the last segment at full contrast", () => {
      renderModal(<Modal.Header>Channel.Create</Modal.Header>);
      expect(screen.getByText("Channel").className).toContain("lyra--color-9");
      expect(screen.getByText("Create").className).toContain("lyra--color-11");
    });

    it("should frame the icon in a tile, with no separator after it", () => {
      const withIcon = renderModal(
        <Modal.Header icon={<Icon.Add className="test-icon" />}>Title</Modal.Header>,
      );
      expect(
        withIcon.baseElement.querySelector(".lyra-icon-tile > .test-icon"),
      ).not.toBeNull();
      expect(
        withIcon.baseElement.querySelector(".lyra-breadcrumb__separator"),
      ).toBeNull();
      withIcon.unmount();
      const withoutIcon = renderModal(<Modal.Header>Title</Modal.Header>);
      expect(withoutIcon.baseElement.querySelector(".test-icon")).toBeNull();
    });

    it("should show the badge after the breadcrumb", () => {
      renderModal(
        <Modal.Header badge={<span>USB</span>}>{["Configure", "T7"]}</Modal.Header>,
      );
      const title = screen.getByText("T7");
      const badge = screen.getByText("USB");
      expect(badge.closest(".lyra-breadcrumb")).toBeNull();
      expect(
        title.compareDocumentPosition(badge) & Node.DOCUMENT_POSITION_FOLLOWING,
      ).toBeTruthy();
    });

    it("should name the close action and its escape key in the tooltip", async () => {
      renderModal(
        <Tooltip.Config delay={0}>
          <Modal.Header>Title</Modal.Header>
        </Tooltip.Config>,
      );
      const close = screen.getByLabelText("Close");
      close.getBoundingClientRect = () => new DOMRect(100, 100, 24, 24);
      fireEvent.pointerOver(close, { pointerType: "mouse" });
      const tip = await waitFor(() => {
        const el = document.querySelector<HTMLElement>(".lyra-tooltip");
        if (el == null) throw new Error("tooltip did not open");
        return el;
      });
      expect(tip.textContent).toBe("CloseEsc");
    });

    it("should close the enclosing dialog from the close button", () => {
      const { onVisibleChange } = renderModal(<Modal.Header>Title</Modal.Header>);
      fireEvent.click(screen.getByLabelText("Close"));
      expect(onVisibleChange).toHaveBeenCalledWith(false);
    });

    it("should omit the close button when hidden", () => {
      renderModal(<Modal.Header closeHidden>Title</Modal.Header>);
      expect(screen.queryByLabelText("Close")).toBeNull();
    });
  });

  describe("Body", () => {
    it("should render children inside the modal body element", () => {
      const { baseElement } = renderModal(
        <Modal.Body className="extra">body content</Modal.Body>,
      );
      const el = baseElement.querySelector(".lyra-modal__body");
      expect(el?.className).toContain("extra");
      expect(screen.getByText("body content")).toBeTruthy();
    });

    it("should stack its children one field gap apart by default", () => {
      const { baseElement } = renderModal(<Modal.Body>body</Modal.Body>);
      const el = baseElement.querySelector(".lyra-modal__body");
      expect(el?.className).toContain("lyra--field-gap");
    });

    it.each([
      ["gap", { gap: "small" }],
      ["pack", { pack: true }],
      ["empty", { empty: true }],
    ] as const)(
      "should leave the spacing to %s when the caller sets it",
      (_, props) => {
        const { baseElement } = renderModal(<Modal.Body {...props}>body</Modal.Body>);
        const el = baseElement.querySelector(".lyra-modal__body");
        expect(el?.className).not.toContain("lyra--field-gap");
      },
    );
  });

  describe("Footer", () => {
    it("should render children inside the modal footer element", () => {
      const { baseElement } = renderModal(
        <Modal.Footer className="extra">footer content</Modal.Footer>,
      );
      const el = baseElement.querySelector(".lyra-modal__footer");
      expect(el?.className).toContain("extra");
      expect(screen.getByText("footer content")).toBeTruthy();
    });
  });
});
