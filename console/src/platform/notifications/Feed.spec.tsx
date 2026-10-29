// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type status } from "@synnaxlabs/client";
import { Status } from "@synnaxlabs/lyra/status";
import { fireEvent, screen } from "@testing-library/react";
import { type ReactElement } from "react";
import { describe, expect, it } from "vitest";

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
});
