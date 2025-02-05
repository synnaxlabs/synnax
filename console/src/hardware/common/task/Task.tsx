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
import { type FC, type ReactElement } from "react";
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
};

export interface TaskProps<
  C extends UnknownRecord = UnknownRecord,
  D extends {} = UnknownRecord,
  T extends string = string,
> {
  task: task.Task<C, D, T> | task.Payload<C, D, T>;
  layoutKey: string;
}

export type ConfigSchema<C extends UnknownRecord = UnknownRecord> = z.ZodType<
  C,
  z.ZodTypeDef,
  unknown
>;

export interface ZeroPayloadFunction<
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
  zeroPayload: task.Payload<C, D, T> | ZeroPayloadFunction<C, D, T>;
  configSchema?: ConfigSchema<C>;
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
    const client = Synnax.use();
    const args = Layout.useSelectArgs<LayoutArgs>(layoutKey);
    const {
      isError,
      error,
      isPending,
      data: task,
    } = useQuery({
      queryKey: [layoutKey, client?.key, args?.taskKey, args?.deviceKey],
      queryFn: async () => {
        const { zeroPayload, configSchema: configurationSchema } = options;
        if (args?.taskKey == null)
          return typeof zeroPayload === "function"
            ? zeroPayload(args?.deviceKey)
            : zeroPayload;

        if (client == null) throw new Error("Synnax server not connected");
        const task = await client.hardware.tasks.retrieve<C, D, T>(args.taskKey, {
          includeState: true,
        });
        if (configurationSchema) task.config = configurationSchema.parse(task.config);
        return task;
      },
    });
    let content: ReactElement | null = null;
    content = isPending ? (
      <Status.Text.Centered variant="loading" level="h2">
        Fetching task from server
      </Status.Text.Centered>
    ) : isError ? (
      <Align.Space direction="y" grow align="center" justify="center">
        <Text.Text level="h2" color={Status.variantColors.error}>
          Failed to load data for task with key {layoutKey}
        </Text.Text>
        <Text.Text level="p" color={Status.variantColors.error}>
          {error.message}
        </Text.Text>
      </Align.Space>
    ) : (
      <Task layoutKey={layoutKey} task={task} />
    );
    return <Eraser.Eraser>{content}</Eraser.Eraser>;
  };
  Wrapper.displayName = `TaskWrapper(${Task.displayName ?? Task.name})`;
  return Wrapper;
};
