// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Status } from "@synnaxlabs/pluto";
import { fireEvent, screen } from "@testing-library/react";
import { type ReactElement } from "react";
import { afterEach, beforeEach, describe, expect, it } from "vitest";

import { Notifications } from "@/platform/notifications";
import { renderWithConsole } from "@/testutil";

interface Seed {
  key: string;
  variant: "info" | "success" | "error";
  message: string;
}

const Harness = ({
  seed,
  adapters = [],
}: {
  seed: Seed;
  adapters?: Notifications.Adapter[];
}): ReactElement => {
  const add = Status.useAdder();
  return (
    <>
      <button onClick={() => add(seed)}>add</button>
      <Notifications.Notifications adapters={adapters} />
    </>
  );
};
Harness.displayName = "Harness";

const addStatus = async (
  seed: Seed,
  adapters?: Notifications.Adapter[],
): Promise<void> => {
  await renderWithConsole(<Harness seed={seed} adapters={adapters} />);
  fireEvent.click(screen.getByText("add"));
};

describe("Notifications", () => {
  beforeEach(() => {
    const root = document.createElement("div");
    root.id = "root";
    document.body.appendChild(root);
  });
  afterEach(() => document.getElementById("root")?.remove());

  it("renders a notification for an ordinary status", async () => {
    await addStatus({ key: "n1", variant: "info", message: "Hello there" });
    expect(screen.getByText("Hello there")).toBeTruthy();
  });

  it("hides repeated rack/device/task success notifications", async () => {
    await addStatus({ key: "rack-1", variant: "success", message: "Rack connected" });
    expect(screen.queryByText("Rack connected")).toBeNull();
  });

  it("still shows rack error notifications", async () => {
    await addStatus({ key: "rack-1", variant: "error", message: "Rack failed" });
    expect(screen.getByText("Rack failed")).toBeTruthy();
  });

  it("applies an adapter's sugared content", async () => {
    const adapter: Notifications.Adapter = (status) =>
      status.key === "special"
        ? { ...status, content: <span>Adapted content</span> }
        : null;
    await addStatus({ key: "special", variant: "info", message: "original" }, [
      adapter,
    ]);
    expect(screen.getByText("Adapted content")).toBeTruthy();
  });
});
