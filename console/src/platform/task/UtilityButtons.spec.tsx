// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { fireEvent, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, type Mock } from "vitest";

import { createClusterState } from "@/platform/cluster/testutil";
import { Task } from "@/platform/task";
import { renderInTaskForm } from "@/platform/task/testutil";
import { getIconButton, queryIcon, stubClipboardWriteText } from "@/testutil";

const clickIcon = (container: HTMLElement, icon: string): void => {
  fireEvent.click(getIconButton(container, icon));
};

describe("UtilityButtons", () => {
  let writeText: Mock;

  beforeEach(() => {
    writeText = stubClipboardWriteText();
  });

  it("should only show the JSON config button when the task has no key", async () => {
    const { container } = await renderInTaskForm(<Task.UtilityButtons />, {
      values: { key: undefined, name: "T", config: {} },
    });
    expect(queryIcon(container, "json")).toBeTruthy();
    expect(queryIcon(container, "typescript")).toBeNull();
    expect(queryIcon(container, "python")).toBeNull();
    expect(queryIcon(container, "link")).toBeNull();
    expect(queryIcon(container, "export")).toBeNull();
  });

  it("should show the code, link, and export buttons when the task has a key", async () => {
    const { container } = await renderInTaskForm(<Task.UtilityButtons />, {
      values: { key: "task_123", name: "T", config: {} },
    });
    expect(queryIcon(container, "json")).toBeTruthy();
    expect(queryIcon(container, "typescript")).toBeTruthy();
    expect(queryIcon(container, "python")).toBeTruthy();
    expect(queryIcon(container, "link")).toBeTruthy();
    expect(queryIcon(container, "export")).toBeTruthy();
  });

  it("should copy TypeScript code that retrieves the task", async () => {
    const { container } = await renderInTaskForm(<Task.UtilityButtons />, {
      values: { key: "task_123", name: "My Task", config: {} },
    });
    clickIcon(container, "typescript");
    await waitFor(() => expect(writeText).toHaveBeenCalledTimes(1));
    const code = writeText.mock.calls[0][0];
    expect(code).toContain('const task = client.tasks.retrieve("task_123")');
    expect(code).toContain("// Retrieve My Task");
  });

  it("should copy Python code that retrieves the task", async () => {
    const { container } = await renderInTaskForm(<Task.UtilityButtons />, {
      values: { key: "task_123", name: "My Task", config: {} },
    });
    clickIcon(container, "python");
    await waitFor(() => expect(writeText).toHaveBeenCalledTimes(1));
    const code = writeText.mock.calls[0][0];
    expect(code).toContain('task = client.tasks.retrieve("task_123")');
    expect(code).toContain("# Retrieve My Task");
  });

  it("should copy the JSON-encoded task configuration", async () => {
    const { container } = await renderInTaskForm(<Task.UtilityButtons />, {
      values: { name: "My Task", config: { device: "dev_1", sampleRate: 25 } },
    });
    clickIcon(container, "json");
    await waitFor(() => expect(writeText).toHaveBeenCalledTimes(1));
    expect(JSON.parse(writeText.mock.calls[0][0])).toEqual({
      channels: [],
      device: "dev_1",
      sample_rate: 25,
    });
  });

  it("should copy a link containing the active cluster and the task key", async () => {
    const { container } = await renderInTaskForm(<Task.UtilityButtons />, {
      values: { key: "task_123", name: "My Task", config: {} },
      preloadedState: createClusterState([], "cluster_1"),
    });
    clickIcon(container, "link");
    await waitFor(() => expect(writeText).toHaveBeenCalledTimes(1));
    const url = writeText.mock.calls[0][0];
    expect(url).toContain("cluster_1");
    expect(url).toContain("task");
    expect(url).toContain("task_123");
  });
});
