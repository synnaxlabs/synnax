// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ReactNode } from "react";
import { describe, expect, it } from "vitest";
import { page } from "vitest/browser";

import { Button } from "@/button";
import { CSS } from "@/css";
import { Dialog } from "@/dialog";
import { Modal } from "@/modal";
import { Nav } from "@/nav";
import { computed, renderThemed } from "@/styletest";
import { Triggers } from "@/triggers";

const renderModal = async (body: ReactNode = "Body"): Promise<Element> => {
  await renderThemed(
    <Triggers.Provider>
      <Dialog.Frame variant="modal" visible>
        <Modal.Frame>
          <Modal.Header>Title</Modal.Header>
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
});
