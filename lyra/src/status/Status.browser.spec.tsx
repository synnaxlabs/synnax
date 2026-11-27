// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { TimeStamp } from "@synnaxlabs/x";
import { assert, describe, expect, it } from "vitest";
import { userEvent } from "vitest/browser";

import { CSS } from "@/css";
import { Status } from "@/status";
import { computed, renderThemed, resolve, withMotion } from "@/styletest";

const MESSAGE = "Failed to start the task because the device did not answer in time";

const renderNotification = async (): Promise<Element> => {
  const { getByRole } = await renderThemed(
    <Status.Notification
      status={{
        key: "status",
        name: "Task",
        time: TimeStamp.now(),
        count: 1,
        message: MESSAGE,
        description: "Check the device connection.",
        variant: "error",
      }}
      silence={() => {}}
    />,
  );
  return getByRole("status").element();
};

describe("Status", () => {
  it.each([
    ["error", "--pluto-error-z"],
    ["warning", "--pluto-warning-z"],
    ["success", "--pluto-primary-z"],
    ["disabled", "--pluto-text-disabled"],
  ] as const)("should color %s text with its variant token", async (variant, token) => {
    const { getByText } = await renderThemed(
      <Status.Summary variant={variant} message="Message" hideIcon />,
    );
    expect(computed(getByText("Message").element(), "color")).toBe(
      resolve(`var(${token})`),
    );
  });

  describe("summary with a description", () => {
    const DESCRIPTION =
      "Line plots need a GPU to render. Run the Console on a machine with a GPU. ".repeat(
        3,
      );

    const renderSummary = async (center?: boolean) => {
      const { getByText } = await renderThemed(
        <div style={{ width: "1200px", height: "400px", display: "flex" }}>
          <Status.Summary
            variant="error"
            message="GPU required"
            description={DESCRIPTION}
            center={center}
          />
        </div>,
      );
      const description = getByText(DESCRIPTION.trim()).element();
      const box = description.parentElement!.getBoundingClientRect();
      return { description, box, rect: description.getBoundingClientRect() };
    };

    it("should center the description and wrap it at 80rem in a wide box", async () => {
      const { description, box, rect } = await renderSummary();
      expect(computed(description, "text-align")).toBe("center");
      expect(rect.width).toBe(Number.parseFloat(resolve("80rem", "width")));
      expect(rect.left - box.left).toBeCloseTo(box.right - rect.right, 0);
    });

    it("should start the description at the left edge when not centered", async () => {
      const { description, box, rect } = await renderSummary(false);
      expect(computed(description, "text-align")).toBe("start");
      expect(rect.left).toBe(box.left);
    });
  });

  describe("notification", () => {
    it("should float on the lift shadow with a border", async () => {
      const card = await renderNotification();
      expect(computed(card, "box-shadow")).toBe(
        resolve("var(--pluto-shadow-lift)", "box-shadow"),
      );
      expect(computed(card, "border-top-color")).toBe(resolve("var(--pluto-gray-l6)"));
    });

    it("should wrap a long message inside the card", async () => {
      const card = await renderNotification();
      const message = card.querySelector(`.${CSS.BE("notification", "message")}`)!;
      expect(message.getBoundingClientRect().right).toBeLessThanOrEqual(
        card.getBoundingClientRect().right,
      );
    });

    it("should swap its time for its controls on hover", async () => {
      const card = await renderNotification();
      const stamp = card.querySelector(`.${CSS.BE("notification", "stamp")}`)!;
      const controls = card.querySelector(`.${CSS.BE("notification", "controls")}`)!;
      expect(computed(stamp, "opacity")).toBe("1");
      expect(computed(controls, "opacity")).toBe("0");
      await userEvent.hover(card);
      expect(computed(stamp, "opacity")).toBe("0");
      expect(computed(controls, "opacity")).toBe("1");
    });
  });

  it("should play its entrance once, not each time a parent shows it again", async () => {
    const card = await renderNotification();
    card.dispatchEvent(new AnimationEvent("animationend", { bubbles: true }));
    const host = card.parentElement;
    assert(host != null);
    await expect.poll(() => card.className).not.toContain(CSS.M("entering"));
    const replays = withMotion(() => {
      host.style.display = "none";
      host.getBoundingClientRect();
      host.style.display = "";
      host.getBoundingClientRect();
      return card.getAnimations().length;
    });
    expect(replays).toBe(0);
  });
});
