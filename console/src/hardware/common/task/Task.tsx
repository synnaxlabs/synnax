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
  Config extends UnknownRecord = UnknownRecord,
  Details extends {} = UnknownRecord,
  Type extends string = string,
> {
  layoutKey: string;
  task: task.Payload<Config, Details, Type> | task.Task<Config, Details, Type>;
}

export interface ConfigSchema<Config extends UnknownRecord = UnknownRecord>
  extends z.ZodType<Config, z.ZodTypeDef, unknown> {}

export interface GetInitialPayload<
  Config extends UnknownRecord = UnknownRecord,
  Details extends {} = UnknownRecord,
  Type extends string = string,
> {
  (deviceKey?: device.Key): task.Payload<Config, Details, Type>;
}

export interface WrapOptions<
  Config extends UnknownRecord = UnknownRecord,
  Details extends {} = UnknownRecord,
  Type extends string = string,
> {
  configSchema: ConfigSchema<Config>;
  getInitialPayload: GetInitialPayload<Config, Details, Type>;
}

export const wrap = <
  Config extends UnknownRecord = UnknownRecord,
  Details extends {} = UnknownRecord,
  Type extends string = string,
>(
  Task: FC<TaskProps<Config, Details, Type>>,
  options: WrapOptions<Config, Details, Type>,
): Layout.Renderer => {
  const Wrapper: Layout.Renderer = ({ layoutKey }) => {
    const { deviceKey, taskKey } = Layout.useSelectArgs<LayoutArgs>(layoutKey);
    const client = Synnax.use();
    const { data, error, isError, isPending } = useQuery({
      queryFn: async () => {
        const { configSchema, getInitialPayload } = options;
        if (taskKey == null) return getInitialPayload(deviceKey);
        if (client == null) throw new Error("Synnax server not connected");
        const task = await client.hardware.tasks.retrieve<Config, Details, Type>(
          taskKey,
          { includeState: true },
        );
        task.config = configSchema.parse(task.config);
        return task;
      },
      queryKey: [taskKey, deviceKey, client?.key, layoutKey],
    });
    const content = isPending ? (
      <Status.Text.Centered level="h4" variant="loading">
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
