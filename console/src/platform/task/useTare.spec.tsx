// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { DisconnectedError, type Synnax, task } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { act, renderHook } from "@testing-library/react";
import { type FC, type PropsWithChildren, type ReactElement } from "react";
import { describe, expect, it } from "vitest";

import { Task } from "@/platform/task";
import {
  awaitCommand,
  TaskFormProvider,
  type TaskFormValues,
} from "@/platform/task/testutil";
import { createConsoleWrapper, uniqueName } from "@/testutil";

interface Chan extends Task.TareableChannel {}

const client = createTestClient();
const rack = await client.racks.create({ name: uniqueName("rack") });

const createTask = async () =>
  await rack.createTask({ name: uniqueName("tsk"), type: "test_type", config: {} });

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
      const { result } = await renderUseTare(
        { status: { details: { running: true } } },
        { client },
      );
      expect(() => result.current[0](10)).toThrow("Task has not been configured");
    });

    it("should execute a tare command for a single channel", async () => {
      const tsk = await createTask();
      const streamer = await client.openStreamer(task.COMMAND_CHANNEL_NAME);
      try {
        const { result } = await renderUseTare(
          { key: tsk.key, status: { details: { running: true } } },
          { client },
        );
        await act(async () => {
          result.current[0](7);
        });
        const cmd = await awaitCommand(streamer, tsk.key);
        expect(cmd.type).toBe("tare");
        expect(cmd.args).toEqual({ keys: [7] });
      } finally {
        streamer.close();
      }
    });
  });

  describe("handleTare", () => {
    it("should tare only the tareable channels matching the selected keys", async () => {
      const tsk = await createTask();
      const streamer = await client.openStreamer(task.COMMAND_CHANNEL_NAME);
      try {
        const { result } = await renderUseTare(
          { key: tsk.key, status: { details: { running: true } } },
          { client },
        );
        await act(async () => {
          result.current[2](["a", "b"], CHANNELS);
        });
        const cmd = await awaitCommand(streamer, tsk.key);
        expect(cmd.type).toBe("tare");
        expect(cmd.args).toEqual({ keys: [10, 20] });
      } finally {
        streamer.close();
      }
    });
  });
});
