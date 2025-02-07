// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type device, type task } from "@synnaxlabs/client";
import { Align, Eraser, Status, Synnax, Text } from "@synnaxlabs/pluto";
import { type UnknownRecord } from "@synnaxlabs/x";
import { useQuery } from "@tanstack/react-query";
import { type FC } from "react";
import { type z } from "zod";

import { Layout } from "@/layout";

interface LayoutArgs {
  taskKey?: task.Key;
  deviceKey?: device.Key;
}

export interface LayoutBaseState extends Layout.BaseState<LayoutArgs> {}

export const LAYOUT: Omit<LayoutBaseState, "type" | "key"> = {
  name: "Configure",
  icon: "Task",
  location: "mosaic",
  args: {},
};

export interface TaskProps<
  C extends UnknownRecord = UnknownRecord,
  D extends {} = UnknownRecord,
  T extends string = string,
> {
  layoutKey: string;
  task: task.Payload<C, D, T> | task.Task<C, D, T>;
}

export interface ConfigSchema<C extends UnknownRecord = UnknownRecord>
  extends z.ZodType<C, z.ZodTypeDef, unknown> {}

export interface GetInitialPayload<
  C extends UnknownRecord = UnknownRecord,
  D extends {} = UnknownRecord,
  T extends string = string,
> {
  (deviceKey?: device.Key): task.Payload<C, D, T>;
}

export interface WrapOptions<
  C extends UnknownRecord = UnknownRecord,
  D extends {} = UnknownRecord,
  T extends string = string,
> {
  configSchema: ConfigSchema<C>;
  getInitialPayload: GetInitialPayload<C, D, T>;
}

export const wrap = <
  C extends UnknownRecord = UnknownRecord,
  D extends {} = UnknownRecord,
  T extends string = string,
>(
  Task: FC<TaskProps<C, D, T>>,
  options: WrapOptions<C, D, T>,
): Layout.Renderer => {
  const Wrapper: Layout.Renderer = ({ layoutKey }) => {
    const { deviceKey, taskKey } = Layout.useSelectArgs<LayoutArgs>(layoutKey);
    const client = Synnax.use();
    const { data, error, isError, isPending } = useQuery({
      queryFn: async () => {
        const { configSchema, getInitialPayload } = options;
        if (taskKey == null) return getInitialPayload(deviceKey);
        if (client == null) throw new Error("Synnax server not connected");
        const task = await client.hardware.tasks.retrieve<C, D, T>(taskKey, {
          includeState: true,
        });
        task.config = configSchema.parse(task.config);
        return task;
      },
      queryKey: [taskKey, deviceKey, client?.key, layoutKey],
    });
    const content = isPending ? (
      <Status.Text.Centered level="h2" variant="loading">
        Fetching task from server
      </Status.Text.Centered>
    ) : isError ? (
      <Align.Space align="center" grow justify="center">
        <Text.Text color={Status.variantColors.error} level="h2">
          Failed to load data for task with key {layoutKey}
        </Text.Text>
        <Text.Text color={Status.variantColors.error} level="p">
          {error.message}
        </Text.Text>
      </Align.Space>
    ) : (
      <Task layoutKey={layoutKey} task={data} />
    );
    return <Eraser.Eraser>{content}</Eraser.Eraser>;
  };
  Wrapper.displayName = `TaskWrapper(${Task.displayName ?? Task.name})`;
  return Wrapper;
};
