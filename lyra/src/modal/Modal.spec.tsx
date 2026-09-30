// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { fireEvent, render, screen } from "@testing-library/react";
import { type ReactNode } from "react";
import { describe, expect, it, vi } from "vitest";

import { Dialog } from "@/dialog";
import { Modal } from "@/modal";
import { Triggers } from "@/triggers";

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

describe("Modal", () => {
  describe("Frame", () => {
    it("should render children inside the modal element and forward className", () => {
      const { baseElement } = renderModal(
        <Modal.Frame className="extra">
          <span>frame body</span>
        </Modal.Frame>,
      );
      const el = baseElement.querySelector(".pluto-modal");
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

    it("should close the enclosing dialog from the close button", () => {
      const { onVisibleChange } = renderModal(<Modal.Header>Title</Modal.Header>);
      fireEvent.click(screen.getByLabelText("Close"));
      expect(onVisibleChange).toHaveBeenCalledWith(false);
    });

    it("should omit the close button when hidden", () => {
      renderModal(<Modal.Header hideClose>Title</Modal.Header>);
      expect(screen.queryByLabelText("Close")).toBeNull();
    });
  });

  describe("Body", () => {
    it("should render children inside the modal body element", () => {
      const { baseElement } = renderModal(
        <Modal.Body className="extra">body content</Modal.Body>,
      );
      const el = baseElement.querySelector(".pluto-modal__body");
      expect(el?.className).toContain("extra");
      expect(screen.getByText("body content")).toBeTruthy();
    });
  });

  describe("Footer", () => {
    it("should render children inside the modal footer element", () => {
      const { baseElement } = renderModal(
        <Modal.Footer className="extra">footer content</Modal.Footer>,
      );
      const el = baseElement.querySelector(".pluto-modal__footer");
      expect(el?.className).toContain("extra");
      expect(screen.getByText("footer content")).toBeTruthy();
    });
  });
});
