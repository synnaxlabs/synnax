// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type status } from "@synnaxlabs/client";
import { Status } from "@synnaxlabs/pluto";
import { fireEvent, screen } from "@testing-library/react";
import { type ReactElement } from "react";
import { afterEach, beforeEach, describe, expect, it } from "vitest";

import { Notifications } from "@/platform/notifications";
import { renderWithConsole } from "@/testutil";

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

const Suppressed = Notifications.createSuppressed((status) => status.key === "hidden");

describe("Notifications", () => {
  beforeEach(() => {
    const root = document.createElement("div");
    root.id = "root";
    document.body.appendChild(root);
  });
  afterEach(() => document.getElementById("root")?.remove());

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

  it("suppresses a status when its matching component renders nothing", async () => {
    await addStatus({ key: "hidden", variant: "info", message: "Should not appear" }, [
      Suppressed,
    ]);
    expect(screen.queryByText("Should not appear")).toBeNull();
  });

  describe("matchVariants", () => {
    const loadingOrSuccess = Notifications.createSuppressed(
      Notifications.matchVariants("loading", "success"),
    );

    it("matches a status whose variant is in the list", async () => {
      await addStatus({ key: "n1", variant: "success", message: "Done" }, [
        loadingOrSuccess,
      ]);
      expect(screen.queryByText("Done")).toBeNull();
    });

    it("does not match a status whose variant is absent from the list", async () => {
      await addStatus({ key: "n1", variant: "error", message: "Failed" }, [
        loadingOrSuccess,
      ]);
      expect(screen.getByText("Failed")).toBeTruthy();
    });
  });

  describe("matchPrefix", () => {
    const rackPrefix = Notifications.createSuppressed(
      Notifications.matchPrefix("rack"),
    );

    it("matches a status whose key starts with the prefix", async () => {
      await addStatus({ key: "rack.heartbeat", variant: "info", message: "Rack" }, [
        rackPrefix,
      ]);
      expect(screen.queryByText("Rack")).toBeNull();
    });

    it("does not match a status whose key lacks the prefix", async () => {
      await addStatus({ key: "device.heartbeat", variant: "info", message: "Device" }, [
        rackPrefix,
      ]);
      expect(screen.getByText("Device")).toBeTruthy();
    });
  });

  describe("composeMatchers", () => {
    const both = Notifications.createSuppressed(
      Notifications.composeMatchers(
        Notifications.matchPrefix("rack"),
        Notifications.matchVariants("success"),
      ),
    );

    it("matches when every composed matcher matches", async () => {
      await addStatus(
        { key: "rack.heartbeat", variant: "success", message: "Rack ok" },
        [both],
      );
      expect(screen.queryByText("Rack ok")).toBeNull();
    });

    it("does not match when only some composed matchers match", async () => {
      await addStatus(
        { key: "rack.heartbeat", variant: "error", message: "Rack failed" },
        [both],
      );
      expect(screen.getByText("Rack failed")).toBeTruthy();
    });

    it("matches everything when composed with no matchers", async () => {
      const vacuous = Notifications.createSuppressed(Notifications.composeMatchers());
      await addStatus({ key: "anything", variant: "info", message: "Anything" }, [
        vacuous,
      ]);
      expect(screen.queryByText("Anything")).toBeNull();
    });
  });

  describe("createSuppressRoutineForPrefix", () => {
    const rackRoutine = Notifications.createSuppressRoutineForPrefix("rack");

    it("suppresses a loading status with a matching key prefix", async () => {
      await addStatus(
        { key: "rack.heartbeat", variant: "loading", message: "Heartbeat pending" },
        [rackRoutine],
      );
      expect(screen.queryByText("Heartbeat pending")).toBeNull();
    });

    it("suppresses a success status with a matching key prefix", async () => {
      await addStatus(
        { key: "rack.heartbeat", variant: "success", message: "Heartbeat ok" },
        [rackRoutine],
      );
      expect(screen.queryByText("Heartbeat ok")).toBeNull();
    });

    it("does not suppress a matching prefix with a non-routine variant", async () => {
      await addStatus(
        { key: "rack.heartbeat", variant: "error", message: "Heartbeat failed" },
        [rackRoutine],
      );
      expect(screen.getByText("Heartbeat failed")).toBeTruthy();
    });

    it("does not suppress a routine variant with a different key prefix", async () => {
      await addStatus(
        { key: "device.heartbeat", variant: "success", message: "Device ok" },
        [rackRoutine],
      );
      expect(screen.getByText("Device ok")).toBeTruthy();
    });
  });

  describe("overflow", () => {
    const BatchHarness = ({
      crudes,
      notifications = [],
    }: {
      crudes: status.Crude[];
      notifications?: Notifications.Notification[];
    }): ReactElement => {
      const add = Status.useAdder();
      return (
        <>
          <button onClick={() => crudes.forEach((c) => add(c))}>add</button>
          <Notifications.Feed notifications={notifications} />
        </>
      );
    };
    BatchHarness.displayName = "BatchHarness";

    const addStatuses = async (
      crudes: status.Crude[],
      notifications?: Notifications.Notification[],
    ): Promise<void> => {
      await renderWithConsole(
        <BatchHarness crudes={crudes} notifications={notifications} />,
      );
      fireEvent.click(screen.getByText("add"));
    };

    const createCrudes = (count: number, keyPrefix = "n"): status.Crude[] =>
      Array.from({ length: count }, (_, i) => ({
        key: `${keyPrefix}${i + 1}`,
        variant: "info" as const,
        message: `Message ${keyPrefix}${i + 1}`,
      }));

    it("shows only the four newest toasts and an overflow chip", async () => {
      await addStatuses(createCrudes(6));
      expect(screen.getByText("Message n6")).toBeTruthy();
      expect(screen.getByText("Message n3")).toBeTruthy();
      expect(screen.queryByText("Message n2")).toBeNull();
      expect(screen.queryByText("Message n1")).toBeNull();
      expect(screen.getByText("+2 More")).toBeTruthy();
    });

    it("shows no overflow controls at or below the cap", async () => {
      await addStatuses(createCrudes(4));
      expect(screen.getByText("Message n1")).toBeTruthy();
      expect(screen.queryByText(/More/)).toBeNull();
      expect(screen.queryByText("Clear All")).toBeNull();
    });

    it("expands to show every toast and collapses back", async () => {
      await addStatuses(createCrudes(6));
      fireEvent.click(screen.getByText("+2 More"));
      expect(screen.getByText("Message n1")).toBeTruthy();
      fireEvent.click(screen.getByText("Show Less"));
      expect(screen.queryByText("Message n1")).toBeNull();
      expect(screen.getByText("+2 More")).toBeTruthy();
    });

    it("clears every toast, including hidden overflow, from the chip", async () => {
      await addStatuses(createCrudes(6));
      fireEvent.click(screen.getByText("Clear All"));
      expect(screen.queryByText(/Message n/)).toBeNull();
      expect(screen.queryByText("Clear All")).toBeNull();
    });

    it("does not count suppressed statuses toward the cap or overflow", async () => {
      const suppressed = Notifications.createSuppressed(
        Notifications.matchPrefix("hidden"),
      );
      await addStatuses(
        [...createCrudes(2, "hidden"), ...createCrudes(4)],
        [suppressed],
      );
      expect(screen.getByText("Message n1")).toBeTruthy();
      expect(screen.getByText("Message n4")).toBeTruthy();
      expect(screen.queryByText(/More/)).toBeNull();
    });
  });
});
