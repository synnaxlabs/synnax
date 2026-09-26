// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { act, screen } from "@testing-library/react";
import { describe, it } from "vitest";

import { Task } from "@/platform/task";
import { renderInTaskForm } from "@/platform/task/testutil";

interface Entry {
  key: string;
  device: string;
}

interface KeysProps {
  path?: string;
}

const Keys = ({ path }: KeysProps) => (
  <span>{Task.useChannelDeviceKeys(path).join(",") || "none"}</span>
);

const createEntry = (key: string, device: string): Entry => ({ key, device });

describe("useChannelDeviceKeys", () => {
  it("should return the distinct device keys sorted, dropping unset ones", async () => {
    const channels = [
      createEntry("1", "b"),
      createEntry("2", "a"),
      createEntry("3", ""),
      createEntry("4", "b"),
    ];
    await renderInTaskForm(<Keys />, { values: { config: { channels } } });
    await screen.findByText("a,b");
  });

  it("should follow edits to the list", async () => {
    const { form } = await renderInTaskForm(<Keys />, {
      values: { config: { channels: [createEntry("1", "a")] } },
    });
    await screen.findByText("a");
    act(() =>
      form.current?.set("config.channels", [
        createEntry("1", "a"),
        createEntry("2", "c"),
      ]),
    );
    await screen.findByText("a,c");
  });

  it("should read the list at a custom path", async () => {
    await renderInTaskForm(<Keys path="config.endpoints" />, {
      values: { config: { endpoints: [createEntry("1", "z")] } },
    });
    await screen.findByText("z");
  });
});
