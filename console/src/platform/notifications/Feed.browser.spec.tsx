// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import type { status } from "@synnaxlabs/client";
import { Dialog } from "@synnaxlabs/lyra/dialog";
import { usePopover } from "@synnaxlabs/lyra/hooks";
import { Status } from "@synnaxlabs/lyra/status";
import { fireEvent, screen, waitFor } from "@testing-library/react";
import { type ReactElement, useRef, useState } from "react";
import { afterAll, beforeAll, describe, expect, it } from "vitest";
import { userEvent } from "vitest/browser";

import { Notifications } from "@/platform/notifications";
import { installPortalRoot, renderWithConsole } from "@/testutil";

const Harness = ({
  crude,
  notifications = [],
}: {
  crude: status.Crude;
  notifications?: Notifications.Notification[];
}): ReactElement => {
  const add = Status.useAdder();
  return (
    <>
      <button onClick={() => add(crude)}>add</button>
      <Notifications.Feed notifications={notifications} />
    </>
  );
};
Harness.displayName = "Harness";

const addStatus = async (
  crude: status.Crude,
  notifications?: Notifications.Notification[],
): Promise<void> => {
  await renderWithConsole(<Harness crude={crude} notifications={notifications} />);
  fireEvent.click(screen.getByText("add"));
};

const Content: Notifications.Notification = ({ status }) => (
  <span>Custom content for {status.message}</span>
);
Content.match = (status) => status.key === "special";

describe("Notifications", () => {
  installPortalRoot();
  // The feed polls its clock each second, and real hovers update the feed and its
  // tooltips. Both run on the browser's schedule, which no act can wrap.
  let actEnvironment: boolean | undefined;
  beforeAll(() => {
    actEnvironment = globalThis.IS_REACT_ACT_ENVIRONMENT;
    globalThis.IS_REACT_ACT_ENVIRONMENT = false;
  });
  afterAll(() => {
    globalThis.IS_REACT_ACT_ENVIRONMENT = actEnvironment;
  });

  it("renders the default notification for an unmatched status", async () => {
    await addStatus({ key: "n1", variant: "info", message: "Hello there" });
    expect(screen.getByText("Hello there")).toBeTruthy();
  });

  it("renders a matching notification component in place of the default", async () => {
    await addStatus({ key: "special", variant: "info", message: "original" }, [
      Content,
    ]);
    expect(screen.getByText("Custom content for original")).toBeTruthy();
  });

  describe("matchPrefix", () => {
    const Rack: Notifications.Notification = ({ status }) => (
      <span>Rack notification: {status.message}</span>
    );
    Rack.match = Notifications.matchPrefix("rack");

    it("matches a status whose key starts with the prefix", async () => {
      await addStatus({ key: "rack.heartbeat", variant: "info", message: "Rack" }, [
        Rack,
      ]);
      expect(screen.getByText("Rack notification: Rack")).toBeTruthy();
    });

    it("does not match a status whose key lacks the prefix", async () => {
      await addStatus({ key: "device.heartbeat", variant: "info", message: "Device" }, [
        Rack,
      ]);
      expect(screen.getByText("Device")).toBeTruthy();
      expect(screen.queryByText(/Rack notification/)).toBeNull();
    });
  });

  describe("overflow", () => {
    const BatchHarness = ({ crudes }: { crudes: status.Crude[] }): ReactElement => {
      const add = Status.useAdder();
      return (
        <>
          <button onClick={() => crudes.forEach((c) => add(c))}>add</button>
          <Notifications.Feed notifications={[]} />
        </>
      );
    };
    BatchHarness.displayName = "BatchHarness";

    const addStatuses = async (crudes: status.Crude[]): Promise<void> => {
      await renderWithConsole(<BatchHarness crudes={crudes} />);
      fireEvent.click(screen.getByText("add"));
    };

    const createCrudes = (count: number): status.Crude[] =>
      Array.from({ length: count }, (_, i) => ({
        key: `n${i + 1}`,
        variant: "info",
        message: `Message n${i + 1}`,
      }));

    it("shows only the four newest toasts and an overflow chip", async () => {
      await addStatuses(createCrudes(6));
      expect(screen.getByText("Message n6")).toBeTruthy();
      expect(screen.getByText("Message n3")).toBeTruthy();
      expect(screen.queryByText("Message n2")).toBeNull();
      expect(screen.queryByText("Message n1")).toBeNull();
      expect(screen.getByText("+2 more")).toBeTruthy();
    });

    it("shows no overflow controls at or below the cap", async () => {
      await addStatuses(createCrudes(4));
      expect(screen.getByText("Message n1")).toBeTruthy();
      expect(screen.queryByText(/more/)).toBeNull();
      expect(screen.queryByText("Clear all")).toBeNull();
    });

    it("expands to show every toast and collapses back", async () => {
      await addStatuses(createCrudes(6));
      fireEvent.click(screen.getByText("+2 more"));
      expect(screen.getByText("Message n1")).toBeTruthy();
      fireEvent.click(screen.getByText("Show less"));
      expect(screen.queryByText("Message n1")).toBeNull();
      expect(screen.getByText("+2 more")).toBeTruthy();
    });

    it("clears every toast, including hidden overflow, from the chip", async () => {
      await addStatuses(createCrudes(6));
      fireEvent.click(screen.getByText("Clear all"));
      expect(screen.queryByText(/Message n/)).toBeNull();
      expect(screen.queryByText("Clear all")).toBeNull();
    });
  });

  describe("in a modal", () => {
    const Dismissable: Notifications.Notification = ({ status, silence }) => (
      <button onClick={() => silence(status.key)}>{status.message}</button>
    );
    Dismissable.match = () => true;

    const ModalHarness = (): ReactElement => {
      const add = Status.useAdder();
      return (
        <>
          <button onClick={() => add({ key: "n1", variant: "info", message: "Hello" })}>
            add
          </button>
          <Dialog.Frame variant="modal">
            <Dialog.Trigger>Open</Dialog.Trigger>
            <Dialog.Dialog>Modal</Dialog.Dialog>
          </Dialog.Frame>
          <Notifications.Feed notifications={[Dismissable]} />
        </>
      );
    };
    ModalHarness.displayName = "ModalHarness";

    /** A popover over the whole window, opened after the feed mounts. */
    const Cover = (): ReactElement => {
      const ref = useRef<HTMLDivElement>(null);
      return (
        <>
          <button onClick={() => ref.current?.showPopover()}>cover</button>
          <div
            ref={ref}
            popover="manual"
            style={{ position: "fixed", inset: 0, width: "100vw", height: "100vh" }}
          />
        </>
      );
    };

    /** @returns true when the element is the topmost one at its own center. */
    const onTop = (el: Element): boolean => {
      const { left, top, width, height } = el.getBoundingClientRect();
      const hit = document.elementFromPoint(left + width / 2, top + height / 2);
      return hit != null && el.contains(hit);
    };

    it("shows above an open modal and stays usable there", async () => {
      await renderWithConsole(<ModalHarness />);
      fireEvent.click(screen.getByRole("button", { name: "add" }));
      fireEvent.click(screen.getByRole("button", { name: "Open" }));
      const notification = await screen.findByRole("button", { name: "Hello" });
      expect(screen.getByRole("dialog").contains(notification)).toBe(true);
      expect(onTop(notification)).toBe(true);
      fireEvent.click(notification);
      await waitFor(() => expect(screen.queryByText("Hello")).toBeNull());
      expect(screen.queryByText("Modal")).not.toBeNull();
    });

    /** A popover over the whole window that Lyra shows, as it shows a dropdown. */
    const LyraCover = (): ReactElement => {
      const ref = useRef<HTMLDivElement>(null);
      const [shown, setShown] = useState(false);
      usePopover(ref, shown, null);
      return (
        <>
          <button onClick={() => setShown(true)}>lyra cover</button>
          <div
            ref={ref}
            popover="manual"
            style={{ position: "fixed", inset: 0, width: "100vw", height: "100vh" }}
          />
        </>
      );
    };

    it("keeps a notification over an overlay opened after it", async () => {
      await renderWithConsole(
        <>
          <ModalHarness />
          <LyraCover />
        </>,
      );
      fireEvent.click(screen.getByRole("button", { name: "add" }));
      const notification = await screen.findByRole("button", { name: "Hello" });
      fireEvent.click(screen.getByRole("button", { name: "lyra cover" }));
      await waitFor(() => expect(onTop(notification)).toBe(true));
    });

    it("leaves an overlay opened from the feed over it", async () => {
      await renderWithConsole(
        <>
          <ModalHarness />
          <LyraCover />
        </>,
      );
      fireEvent.click(screen.getByRole("button", { name: "add" }));
      const notification = await screen.findByRole("button", { name: "Hello" });
      notification.focus();
      fireEvent.click(screen.getByRole("button", { name: "lyra cover" }));
      await new Promise(requestAnimationFrame);
      expect(onTop(notification)).toBe(false);
    });

    it("shows a new notification over a popover opened before it", async () => {
      await renderWithConsole(
        <>
          <ModalHarness />
          <Cover />
        </>,
      );
      fireEvent.click(screen.getByRole("button", { name: "cover" }));
      fireEvent.click(screen.getByRole("button", { name: "add" }));
      const notification = await screen.findByRole("button", { name: "Hello" });
      expect(onTop(notification)).toBe(true);
    });

    it("shows again in the top layer after a modal closes", async () => {
      await renderWithConsole(<ModalHarness />);
      fireEvent.click(screen.getByRole("button", { name: "add" }));
      fireEvent.click(screen.getByRole("button", { name: "Open" }));
      await screen.findByText("Modal");
      fireEvent.click(screen.getByRole("button", { name: "Open" }));
      await waitFor(() => expect(screen.queryByText("Modal")).toBeNull());
      const notification = await screen.findByRole("button", { name: "Hello" });
      expect(notification.closest("[popover]")?.matches(":popover-open")).toBe(true);
      expect(onTop(notification)).toBe(true);
    });
  });

  describe("a card tooltip", () => {
    const TooltipHarness = (): ReactElement => {
      const add = Status.useAdder();
      return (
        <>
          <button onClick={() => add({ key: "n1", variant: "info", message: "First" })}>
            first
          </button>
          <button
            onClick={() => add({ key: "n2", variant: "info", message: "Second" })}
          >
            second
          </button>
          <Dialog.Frame variant="modal">
            <Dialog.Trigger>Open</Dialog.Trigger>
            <Dialog.Dialog>Modal</Dialog.Dialog>
          </Dialog.Frame>
          <Notifications.Feed notifications={[]} />
        </>
      );
    };
    TooltipHarness.displayName = "TooltipHarness";

    /**
     * Hovers the Silence button of the lowest card until its tooltip shows, where a
     * card above it can draw over the tooltip.
     */
    const openTooltip = async (statuses: string[]): Promise<HTMLElement> => {
      await renderWithConsole(<TooltipHarness />);
      for (const name of statuses)
        fireEvent.click(screen.getByRole("button", { name }));
      await waitFor(() =>
        expect(screen.getAllByRole("button", { name: "Silence" })).toHaveLength(
          statuses.length,
        ),
      );
      const lowest = screen
        .getAllByRole("button", { name: "Silence" })
        .sort(
          (a, b) => b.getBoundingClientRect().top - a.getBoundingClientRect().top,
        )[0];
      await userEvent.hover(lowest);
      const tooltip = await screen.findByRole("tooltip");
      await waitFor(() => expect(tooltip.matches(":popover-open")).toBe(true));
      return tooltip;
    };

    /** @returns true when the tooltip draws over everything at its center. */
    const tooltipOnTop = (tooltip: HTMLElement): boolean => {
      // The tooltip lets the pointer through, so a hit test needs it to take it.
      tooltip.style.pointerEvents = "auto";
      const { left, top, width, height } = tooltip.getBoundingClientRect();
      const hit = document.elementFromPoint(left + width / 2, top + height / 2);
      tooltip.style.pointerEvents = "";
      return hit != null && tooltip.contains(hit);
    };

    it("stays over the feed when a status arrives", async () => {
      const tooltip = await openTooltip(["first"]);
      fireEvent.click(screen.getByRole("button", { name: "second" }));
      await screen.findByText("Second");
      await new Promise(requestAnimationFrame);
      expect(tooltipOnTop(tooltip)).toBe(true);
    });

    it("leaves the feed showing when a card opens a modal", async () => {
      await openTooltip(["first"]);
      fireEvent.click(screen.getByRole("button", { name: "Open" }));
      await screen.findByText("Modal");
      const feed = document.querySelector(".console-notifications");
      await waitFor(() => expect(feed?.matches(":popover-open")).toBe(true));
    });
  });
});
