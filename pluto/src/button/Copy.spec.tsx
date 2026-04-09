// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { act, fireEvent, render } from "@testing-library/react";
import { type PropsWithChildren } from "react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { Button } from "@/button";
import { Icon } from "@/icon";
import { Status } from "@/status/base";
import { type NotificationSpec } from "@/status/base/Aggregator";

const StatusSpy = ({ onStatuses }: { onStatuses: (s: NotificationSpec[]) => void }) => {
  const { statuses } = Status.useNotifications();
  onStatuses(statuses);
  return null;
};

const wrapper = ({ children }: PropsWithChildren) => (
  <Status.Aggregator>{children}</Status.Aggregator>
);

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

    it("should render text children alongside the copy icon", () => {
      const c = render(<Button.Copy text="hello">Copy me</Button.Copy>);
      expect(c.getByText("Copy me")).toBeTruthy();
      expect(c.getByLabelText("pluto-icon--copy")).toBeTruthy();
    });

    it("should render a custom icon instead of the copy icon", () => {
      const c = render(
        <Button.Copy text="hello">
          <Icon.Python />
        </Button.Copy>,
      );
      expect(c.getByLabelText("pluto-icon--python")).toBeTruthy();
      expect(c.queryByLabelText("pluto-icon--copy")).toBeFalsy();
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

    it("should copy the result of a sync function", async () => {
      const getText = vi.fn(() => "computed text");
      const c = render(<Button.Copy text={getText} />);
      await act(async () => {
        fireEvent.click(c.getByLabelText("pluto-icon--copy"));
      });
      expect(getText).toHaveBeenCalled();
      expect(writeText).toHaveBeenCalledWith("computed text");
    });

    it("should copy the result of an async function", async () => {
      const getText = vi.fn(async () => "async text");
      const c = render(<Button.Copy text={getText} />);
      await act(async () => {
        fireEvent.click(c.getByLabelText("pluto-icon--copy"));
      });
      expect(getText).toHaveBeenCalled();
      expect(writeText).toHaveBeenCalledWith("async text");
    });

    it("should show the check icon after copying", async () => {
      const c = render(<Button.Copy text="hello" />);
      await act(async () => {
        fireEvent.click(c.getByLabelText("pluto-icon--copy"));
      });
      expect(c.getByLabelText("pluto-icon--check")).toBeTruthy();
      expect(c.queryByLabelText("pluto-icon--copy")).toBeFalsy();
    });

    it("should swap a custom icon for the check icon after copying", async () => {
      const c = render(
        <Button.Copy text="hello">
          <Icon.Python />
        </Button.Copy>,
      );
      expect(c.getByLabelText("pluto-icon--python")).toBeTruthy();
      await act(async () => {
        fireEvent.click(c.getByLabelText("pluto-icon--python"));
      });
      expect(c.getByLabelText("pluto-icon--check")).toBeTruthy();
      expect(c.queryByLabelText("pluto-icon--python")).toBeFalsy();
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

    it("should reset a custom icon after the default duration", async () => {
      const c = render(
        <Button.Copy text="hello">
          <Icon.Python />
        </Button.Copy>,
      );
      await act(async () => {
        fireEvent.click(c.getByLabelText("pluto-icon--python"));
      });
      expect(c.getByLabelText("pluto-icon--check")).toBeTruthy();
      await act(async () => {
        vi.advanceTimersByTime(2000);
      });
      expect(c.getByLabelText("pluto-icon--python")).toBeTruthy();
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

    it("should not show the check icon when copying fails", async () => {
      writeText.mockRejectedValue(new Error("Failed"));
      const c = render(<Button.Copy text="hello" />, { wrapper });
      await act(async () => {
        fireEvent.click(c.getByLabelText("pluto-icon--copy"));
      });
      expect(c.getByLabelText("pluto-icon--copy")).toBeTruthy();
      expect(c.queryByLabelText("pluto-icon--check")).toBeFalsy();
    });
  });

  describe("status notification", () => {
    it("should push a success status when successMessage is provided", async () => {
      const spy = vi.fn();
      const c = render(
        <Status.Aggregator>
          <StatusSpy onStatuses={spy} />
          <Button.Copy text="hello" successMessage="Copied!" />
        </Status.Aggregator>,
      );
      await act(async () => {
        fireEvent.click(c.getByLabelText("pluto-icon--copy"));
      });
      const lastCall = spy.mock.lastCall?.[0] as NotificationSpec[];
      expect(lastCall).toHaveLength(1);
      expect(lastCall[0].message).toBe("Copied!");
      expect(lastCall[0].variant).toBe("success");
    });

    it("should resolve successMessage from a function at click time", async () => {
      const spy = vi.fn();
      let name = "Task A";
      const c = render(
        <Status.Aggregator>
          <StatusSpy onStatuses={spy} />
          <Button.Copy text="hello" successMessage={() => `Copied ${name}`} />
        </Status.Aggregator>,
      );
      name = "Task B";
      await act(async () => {
        fireEvent.click(c.getByLabelText("pluto-icon--copy"));
      });
      const lastCall = spy.mock.lastCall?.[0] as NotificationSpec[];
      expect(lastCall).toHaveLength(1);
      expect(lastCall[0].message).toBe("Copied Task B");
    });

    it("should not show the check icon when successMessage is provided", async () => {
      const c = render(
        <Status.Aggregator>
          <Button.Copy text="hello" successMessage="Copied!" />
        </Status.Aggregator>,
      );
      await act(async () => {
        fireEvent.click(c.getByLabelText("pluto-icon--copy"));
      });
      expect(c.getByLabelText("pluto-icon--copy")).toBeTruthy();
      expect(c.queryByLabelText("pluto-icon--check")).toBeFalsy();
    });

    it("should not push a status when successMessage is not provided", async () => {
      const spy = vi.fn();
      const c = render(
        <Status.Aggregator>
          <StatusSpy onStatuses={spy} />
          <Button.Copy text="hello" />
        </Status.Aggregator>,
      );
      await act(async () => {
        fireEvent.click(c.getByLabelText("pluto-icon--copy"));
      });
      const lastCall = spy.mock.lastCall?.[0] as NotificationSpec[];
      expect(lastCall).toHaveLength(0);
    });

    it("should push an error status when clipboard write fails", async () => {
      writeText.mockRejectedValue(new Error("Clipboard denied"));
      const spy = vi.fn();
      const c = render(
        <Status.Aggregator>
          <StatusSpy onStatuses={spy} />
          <Button.Copy text="hello" />
        </Status.Aggregator>,
      );
      await act(async () => {
        fireEvent.click(c.getByLabelText("pluto-icon--copy"));
      });
      const lastCall = spy.mock.lastCall?.[0] as NotificationSpec[];
      expect(lastCall).toHaveLength(1);
      expect(lastCall[0].variant).toBe("error");
      expect(lastCall[0].message).toBe("Clipboard denied");
    });

    it("should push an error status when an async text function rejects", async () => {
      const getText = vi.fn(() => Promise.reject(new Error("Failed to compute")));
      const spy = vi.fn();
      const c = render(
        <Status.Aggregator>
          <StatusSpy onStatuses={spy} />
          <Button.Copy text={getText} />
        </Status.Aggregator>,
      );
      await act(async () => {
        fireEvent.click(c.getByLabelText("pluto-icon--copy"));
      });
      const lastCall = spy.mock.lastCall?.[0] as NotificationSpec[];
      expect(lastCall).toHaveLength(1);
      expect(lastCall[0].variant).toBe("error");
      expect(lastCall[0].message).toBe("Failed to compute");
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
