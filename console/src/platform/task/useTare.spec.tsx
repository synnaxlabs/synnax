// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createTestClient, DisconnectedError, type Synnax } from "@synnaxlabs/client";
import { act, renderHook, waitFor } from "@testing-library/react";
import { type FC, type PropsWithChildren, type ReactElement } from "react";
import { describe, expect, it, vi } from "vitest";

import { Task } from "@/platform/task";
import { TaskFormProvider, type TaskFormValues } from "@/platform/task/testutil";
import { createConsoleWrapper } from "@/testutil";

interface Chan extends Task.TareableChannel {}

const RUNNING_VALUES: TaskFormValues = {
  key: "task-1",
  status: { details: { running: true } },
};

const CHANNELS: Chan[] = [
  { key: "a", channel: 10 },
  { key: "b", channel: 20 },
  { key: "c", channel: 30 },
];

const renderUseTare = async (
  values: TaskFormValues,
  {
    client = null,
    props,
  }: { client?: Synnax | null; props?: Task.UseTareProps<Chan> } = {},
) => {
  const { wrapper: Console } = await createConsoleWrapper({ client });
  const Wrapper: FC<PropsWithChildren> = ({ children }): ReactElement => (
    <Console>
      <TaskFormProvider values={values}>{children}</TaskFormProvider>
    </Console>
  );
  Wrapper.displayName = "Wrapper";
  return renderHook(() => Task.useTare<Chan>(props), { wrapper: Wrapper });
};

describe("useTare", () => {
  describe("allowTare", () => {
    it("should allow taring when running with a matching tareable channel", async () => {
      const { result } = await renderUseTare(RUNNING_VALUES);
      expect(result.current[1](["a"], CHANNELS)).toBe(true);
    });

    it("should not allow taring when the task is not running", async () => {
      const { result } = await renderUseTare({ key: "task-1" });
      expect(result.current[1](["a"], CHANNELS)).toBe(false);
    });

    it("should not allow taring when no channel matches the selected keys", async () => {
      const { result } = await renderUseTare(RUNNING_VALUES);
      expect(result.current[1](["missing"], CHANNELS)).toBe(false);
    });

    it("should exclude channels rejected by isChannelTareable", async () => {
      const { result } = await renderUseTare(RUNNING_VALUES, {
        props: { isChannelTareable: () => false },
      });
      expect(result.current[1](["a"], CHANNELS)).toBe(false);
    });
  });

  describe("tare", () => {
    it("should throw a DisconnectedError when the client is null", async () => {
      const { result } = await renderUseTare(RUNNING_VALUES);
      let err: unknown;
      try {
        result.current[0](10);
      } catch (e) {
        err = e;
      }
      expect(DisconnectedError.matches(err)).toBe(true);
    });

    it("should throw when the task has not been configured", async () => {
      const client = createTestClient();
      const { result } = await renderUseTare(
        { status: { details: { running: true } } },
        { client },
      );
      expect(() => result.current[0](10)).toThrow("Task has not been configured");
    });

    it("should execute a tare command for a single channel", async () => {
      const client = createTestClient();
      const spy = vi.spyOn(client.tasks, "executeCommand").mockResolvedValue(["cmd"]);
      const { result } = await renderUseTare(RUNNING_VALUES, { client });
      await act(async () => {
        result.current[0](7);
      });
      await waitFor(() => expect(spy).toHaveBeenCalledTimes(1));
      expect(spy).toHaveBeenCalledWith({
        task: "task-1",
        type: "tare",
        args: { keys: [7] },
      });
      spy.mockRestore();
    });
  });

  describe("handleTare", () => {
    it("should tare only the tareable channels matching the selected keys", async () => {
      const client = createTestClient();
      const spy = vi.spyOn(client.tasks, "executeCommand").mockResolvedValue(["cmd"]);
      const { result } = await renderUseTare(RUNNING_VALUES, { client });
      await act(async () => {
        result.current[2](["a", "b"], CHANNELS);
      });
      await waitFor(() => expect(spy).toHaveBeenCalledTimes(1));
      expect(spy).toHaveBeenCalledWith({
        task: "task-1",
        type: "tare",
        args: { keys: [10, 20] },
      });
      spy.mockRestore();
    });
  });
});
