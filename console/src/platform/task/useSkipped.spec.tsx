// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { deep } from "@synnaxlabs/x";
import { act, screen, waitFor } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { Task } from "@/platform/task";
import { renderInTaskForm } from "@/platform/task/testutil";
import { type Channel } from "@/platform/task/types";

interface PortChannel extends Channel {
  port: string;
  commanded: boolean;
}

const path = deep.path<{ config: { channels: PortChannel[] } }>();
const portOf = ({ port }: PortChannel) => port;
const isCommanded = ({ commanded }: PortChannel) => commanded;

interface ProbeProps {
  itemKey: string;
  binds?: (channel: PortChannel) => boolean;
}

const Probe = ({ itemKey, binds }: ProbeProps) => (
  <span data-testid="skipped">
    {String(Task.useSkipped(path.config.channels, itemKey, portOf, binds))}
  </span>
);

const ch = (
  key: string,
  disabled: boolean,
  commanded = true,
  port = "ao0",
): PortChannel => ({ key, port, disabled, commanded });

const renderProbe = async (channels: PortChannel[], props: ProbeProps) =>
  await renderInTaskForm(<Probe {...props} />, { values: { config: { channels } } });

const skipped = () => screen.getByTestId("skipped").textContent;

describe("useSkipped", () => {
  it("should skip a disabled channel when another channel claims its port", async () => {
    await renderProbe([ch("a", false), ch("b", true)], { itemKey: "b" });
    await waitFor(() => expect(skipped()).toBe("true"));
  });

  it("should not skip a disabled channel that is alone on its port", async () => {
    await renderProbe([ch("a", false, true, "ao1"), ch("b", true)], { itemKey: "b" });
    await waitFor(() => expect(skipped()).toBe("false"));
  });

  it("should not skip an enabled channel", async () => {
    await renderProbe([ch("a", false), ch("b", false)], { itemKey: "b" });
    await waitFor(() => expect(skipped()).toBe("false"));
  });

  it("should not let a channel that binds rejects claim the port", async () => {
    await renderProbe([ch("a", false, false), ch("b", true)], {
      itemKey: "b",
      binds: isCommanded,
    });
    await waitFor(() => expect(skipped()).toBe("false"));
  });

  it("should never skip a channel that binds rejects", async () => {
    await renderProbe([ch("a", false), ch("b", true, false)], {
      itemKey: "b",
      binds: isCommanded,
    });
    await waitFor(() => expect(skipped()).toBe("false"));
  });

  it("should update when the form changes", async () => {
    const { form } = await renderProbe([ch("a", false), ch("b", true)], {
      itemKey: "b",
    });
    await waitFor(() => expect(skipped()).toBe("true"));
    act(() => form.current?.set(path.config.channels.b.disabled, false));
    await waitFor(() => expect(skipped()).toBe("false"));
  });
});
