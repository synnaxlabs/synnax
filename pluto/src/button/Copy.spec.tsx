// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { act, fireEvent, render } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { Button } from "@/button";

describe("Copy", () => {
  const writeText = vi.fn();

  beforeEach(() => {
    vi.useFakeTimers();
    Object.assign(navigator, {
      clipboard: { writeText },
    });
    writeText.mockResolvedValue(undefined);
  });

  afterEach(() => {
    vi.useRealTimers();
    writeText.mockReset();
  });

  describe("rendering", () => {
    it("should render the copy icon by default", () => {
      const c = render(<Button.Copy text="hello" />);
      expect(c.getByLabelText("pluto-icon--copy")).toBeTruthy();
    });

    it("should render children alongside the icon", () => {
      const c = render(<Button.Copy text="hello">Copy me</Button.Copy>);
      expect(c.getByText("Copy me")).toBeTruthy();
      expect(c.getByLabelText("pluto-icon--copy")).toBeTruthy();
    });
  });

  describe("copying", () => {
    it("should copy the text to the clipboard when clicked", async () => {
      const c = render(<Button.Copy text="hello world" />);
      await act(async () => {
        fireEvent.click(c.getByLabelText("pluto-icon--copy"));
      });
      expect(writeText).toHaveBeenCalledWith("hello world");
    });

    it("should copy the result of a function when text is a function", async () => {
      const getText = vi.fn(() => "computed text");
      const c = render(<Button.Copy text={getText} />);
      await act(async () => {
        fireEvent.click(c.getByLabelText("pluto-icon--copy"));
      });
      expect(getText).toHaveBeenCalled();
      expect(writeText).toHaveBeenCalledWith("computed text");
    });

    it("should show the check icon after copying", async () => {
      const c = render(<Button.Copy text="hello" />);
      await act(async () => {
        fireEvent.click(c.getByLabelText("pluto-icon--copy"));
      });
      expect(c.getByLabelText("pluto-icon--check")).toBeTruthy();
      expect(c.queryByLabelText("pluto-icon--copy")).toBeFalsy();
    });

    it("should reset to the copy icon after the default duration", async () => {
      const c = render(<Button.Copy text="hello" />);
      await act(async () => {
        fireEvent.click(c.getByLabelText("pluto-icon--copy"));
      });
      expect(c.getByLabelText("pluto-icon--check")).toBeTruthy();
      await act(async () => {
        vi.advanceTimersByTime(2000);
      });
      expect(c.getByLabelText("pluto-icon--copy")).toBeTruthy();
      expect(c.queryByLabelText("pluto-icon--check")).toBeFalsy();
    });

    it("should respect custom copiedDuration", async () => {
      const c = render(<Button.Copy text="hello" copiedDuration={500} />);
      await act(async () => {
        fireEvent.click(c.getByLabelText("pluto-icon--copy"));
      });
      expect(c.getByLabelText("pluto-icon--check")).toBeTruthy();
      await act(async () => {
        vi.advanceTimersByTime(400);
      });
      expect(c.getByLabelText("pluto-icon--check")).toBeTruthy();
      await act(async () => {
        vi.advanceTimersByTime(100);
      });
      expect(c.getByLabelText("pluto-icon--copy")).toBeTruthy();
    });
  });

  describe("callbacks", () => {
    it("should call onCopy after successfully copying", async () => {
      const onCopy = vi.fn();
      const c = render(<Button.Copy text="hello" onCopy={onCopy} />);
      await act(async () => {
        fireEvent.click(c.getByLabelText("pluto-icon--copy"));
      });
      expect(onCopy).toHaveBeenCalledTimes(1);
    });

    it("should call onCopyError when clipboard write fails", async () => {
      const error = new Error("Clipboard access denied");
      writeText.mockRejectedValue(error);
      const onCopyError = vi.fn();
      const c = render(<Button.Copy text="hello" onCopyError={onCopyError} />);
      await act(async () => {
        fireEvent.click(c.getByLabelText("pluto-icon--copy"));
      });
      expect(onCopyError).toHaveBeenCalledWith(error);
    });

    it("should not show the check icon when copying fails", async () => {
      writeText.mockRejectedValue(new Error("Failed"));
      const c = render(<Button.Copy text="hello" />);
      await act(async () => {
        fireEvent.click(c.getByLabelText("pluto-icon--copy"));
      });
      expect(c.getByLabelText("pluto-icon--copy")).toBeTruthy();
      expect(c.queryByLabelText("pluto-icon--check")).toBeFalsy();
    });
  });

  describe("button props", () => {
    it("should pass through variant prop", () => {
      const c = render(<Button.Copy text="hello" variant="filled" />);
      expect(c.container.querySelector(".pluto-btn--filled")).toBeTruthy();
    });

    it("should pass through size prop", () => {
      const c = render(<Button.Copy text="hello" size="small" />);
      expect(c.container.querySelector(".pluto--height-small")).toBeTruthy();
    });

    it("should pass through disabled prop", () => {
      const c = render(<Button.Copy text="hello" disabled />);
      expect(c.container.querySelector(".pluto--disabled")).toBeTruthy();
    });
  });
});
