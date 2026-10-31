// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { TimeStamp } from "@synnaxlabs/x";
import { describe, expect, it } from "vitest";
import { userEvent } from "vitest/browser";

import { CSS } from "@/css";
import { Status } from "@/status";
import { computed, renderThemed, resolve } from "@/styletest";

const MESSAGE = "Failed to start the task because the device did not answer in time";

const renderNotification = async (): Promise<Element> => {
  const { getByRole } = await renderThemed(
    <Status.Notification
      status={{
        id: "notification",
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
});
