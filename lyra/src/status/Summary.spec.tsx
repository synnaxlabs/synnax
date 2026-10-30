// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { render } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { Status } from "@/status";

describe("Summary", () => {
  describe("message", () => {
    it("should render the message in a paragraph by default", () => {
      const c = render(<Status.Summary variant="info" message="Connected" />);
      const el = c.getByText("Connected");
      expect(el.tagName).toBe("P");
      expect(el.className).toContain("lyra-text--p");
      expect(el.className).toContain("lyra-status__text");
    });

    it("should render children in place of the message", () => {
      const c = render(
        <Status.Summary variant="info" message="Connected">
          Custom
        </Status.Summary>,
      );
      expect(c.getByText("Custom")).toBeTruthy();
      expect(c.queryByText("Connected")).toBeNull();
    });

    it("should render the message at the given level", () => {
      const c = render(
        <Status.Summary variant="info" message="Connected" level="h4" />,
      );
      const el = c.getByText("Connected");
      expect(el.tagName).toBe("H4");
      expect(el.className).toContain("lyra-text--h4");
    });

    it("should tint the message with the variant color", () => {
      const c = render(<Status.Summary variant="error" message="Failed" />);
      expect(c.getByText("Failed").className).toContain("lyra--status-error");
    });

    it("should add a custom class to the message", () => {
      const c = render(
        <Status.Summary variant="info" message="Connected" className="custom" />,
      );
      expect(c.getByText("Connected").className).toContain("custom");
    });

    it("should forward extra props to the message", () => {
      const c = render(
        <Status.Summary variant="info" message="Connected" data-testid="summary" />,
      );
      expect(c.getByTestId("summary")).toBe(c.getByText("Connected"));
    });
  });

  describe("indicator", () => {
    it("should render the concentric indicator inside the message", () => {
      const c = render(<Status.Summary variant="success" message="Done" />);
      expect(
        c.getByText("Done").querySelector(".lyra-icon--status-concentric"),
      ).not.toBeNull();
    });

    it("should render a spinner for the loading variant", () => {
      const c = render(<Status.Summary variant="loading" message="Loading" />);
      const el = c.getByText("Loading");
      expect(el.querySelector(".lyra-icon--loading")).not.toBeNull();
      expect(el.querySelector(".lyra-icon--status-concentric")).toBeNull();
    });

    it("should not render an indicator when hideIcon is set", () => {
      const c = render(<Status.Summary variant="success" message="Done" hideIcon />);
      expect(c.container.querySelector(".lyra-icon")).toBeNull();
    });
  });

  describe("description", () => {
    it("should render the description as small text below the message", () => {
      const c = render(
        <Status.Summary variant="error" message="Failed" description="Disk is full" />,
      );
      const description = c.getByText("Disk is full");
      expect(description.className).toContain("lyra-text--small");
      const message = c.getByText("Failed");
      expect(message.parentElement).toBe(description.parentElement);
      expect(message.className).not.toContain("lyra-status__text");
    });

    it("should stack the message and description in a vertical box", () => {
      const c = render(
        <Status.Summary variant="error" message="Failed" description="Disk is full" />,
      );
      const box = c.getByText("Failed").parentElement;
      expect(box?.className).toContain("lyra--direction-y");
    });

    it("should forward extra props to the box instead of the message", () => {
      const c = render(
        <Status.Summary
          variant="error"
          message="Failed"
          description="Disk is full"
          data-testid="summary"
        />,
      );
      expect(c.getByTestId("summary")).toBe(c.getByText("Failed").parentElement);
    });

    it("should keep the custom class on the message", () => {
      const c = render(
        <Status.Summary
          variant="error"
          message="Failed"
          description="Disk is full"
          className="custom"
        />,
      );
      expect(c.getByText("Failed").className).toContain("custom");
    });

    it("should render the message at the given level", () => {
      const c = render(
        <Status.Summary
          variant="error"
          message="Failed"
          description="Disk is full"
          level="h4"
        />,
      );
      expect(c.getByText("Failed").tagName).toBe("H4");
    });

    it("should not render a box when the description is empty", () => {
      const c = render(
        <Status.Summary variant="error" message="Failed" description="" />,
      );
      expect(c.container.firstElementChild).toBe(c.getByText("Failed"));
    });
  });

  describe("status", () => {
    it("should render the message, variant, and description of the status", () => {
      const status = Status.create({
        variant: "error",
        message: "Failed",
        description: "Disk is full",
      });
      const c = render(<Status.Summary status={status} />);
      expect(c.getByText("Failed").className).toContain("lyra--status-error");
      expect(c.getByText("Disk is full")).toBeTruthy();
    });

    it("should prefer the fields of the status over individual props", () => {
      const status = Status.create({ variant: "error", message: "Failed" });
      const c = render(
        <Status.Summary status={status} variant="success" message="Done" />,
      );
      const el = c.getByText("Failed");
      expect(el.className).toContain("lyra--status-error");
      expect(c.queryByText("Done")).toBeNull();
    });

    it("should render the message at the given level", () => {
      const status = Status.create({ variant: "error", message: "Failed" });
      const c = render(<Status.Summary status={status} level="h4" />);
      const el = c.getByText("Failed");
      expect(el.tagName).toBe("H4");
      expect(el.className).toContain("lyra-text--h4");
    });

    it("should render the message at the given level with a description", () => {
      const status = Status.create({
        variant: "error",
        message: "Failed",
        description: "Disk is full",
      });
      const c = render(<Status.Summary status={status} level="small" />);
      expect(c.getByText("Failed").className).toContain("lyra-text--small");
    });

    it("should add a custom class to the message", () => {
      const status = Status.create({ variant: "error", message: "Failed" });
      const c = render(<Status.Summary status={status} className="custom" />);
      expect(c.getByText("Failed").className).toContain("custom");
    });

    it("should not render an indicator when hideIcon is set", () => {
      const status = Status.create({ variant: "error", message: "Failed" });
      const c = render(<Status.Summary status={status} hideIcon />);
      expect(c.container.querySelector(".lyra-icon")).toBeNull();
    });

    it("should render children in place of the status message", () => {
      const status = Status.create({ variant: "error", message: "Failed" });
      const c = render(<Status.Summary status={status}>Custom</Status.Summary>);
      expect(c.getByText("Custom").className).toContain("lyra--status-error");
      expect(c.queryByText("Failed")).toBeNull();
    });

    it("should forward extra props to the message", () => {
      const status = Status.create({ variant: "error", message: "Failed" });
      const c = render(<Status.Summary status={status} data-testid="summary" />);
      expect(c.getByTestId("summary")).toBe(c.getByText("Failed"));
    });

    it("should not leak the other status fields onto the DOM", () => {
      const status = Status.fromException(new Error("Disk is full"), "Failed");
      const c = render(<Status.Summary status={{ ...status, name: "disk" }} />);
      const box = c.getByText("Failed").parentElement;
      expect(box?.hasAttribute("name")).toBe(false);
      expect(box?.hasAttribute("time")).toBe(false);
      expect(box?.hasAttribute("details")).toBe(false);
    });
  });
});
