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
import { afterEach, beforeEach, describe, expect, it } from "vitest";

import { Task } from "@/feature/task";
import { Notifications } from "@/platform/notifications";
import { renderWithConsole } from "@/testutil";

interface HarnessProps {
  crude: status.Crude;
}

const Harness = ({ crude }: HarnessProps) => {
  const add = Status.useAdder();
  return (
    <>
      <button onClick={() => add(crude)}>add</button>
      <Notifications.Feed notifications={Task.NOTIFICATIONS} />
    </>
  );
};
Harness.displayName = "Harness";

const addStatus = async (crude: status.Crude): Promise<void> => {
  await renderWithConsole(<Harness crude={crude} />);
  fireEvent.click(screen.getByText("add"));
};

describe("task notifications", () => {
  beforeEach(() => {
    const root = document.createElement("div");
    root.id = "root";
    document.body.appendChild(root);
  });
  afterEach(() => document.getElementById("root")?.remove());

  it("suppresses a routine loading status for a task", async () => {
    await addStatus({ key: "task:1", variant: "loading", message: "Task starting" });
    expect(screen.queryByText("Task starting")).toBeNull();
  });

  it("suppresses a routine success status for a task", async () => {
    await addStatus({ key: "task:1", variant: "success", message: "Task running" });
    expect(screen.queryByText("Task running")).toBeNull();
  });

  it("does not suppress a task error status", async () => {
    await addStatus({ key: "task:1", variant: "error", message: "Task failed" });
    expect(screen.getByText("Task failed")).toBeTruthy();
  });

  it("does not suppress a routine status for a non-task key", async () => {
    await addStatus({ key: "rack:1", variant: "success", message: "Rack ok" });
    expect(screen.getByText("Rack ok")).toBeTruthy();
  });
});
