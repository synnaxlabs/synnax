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
      <Notifications.Notifications notifications={notifications} />
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
});
