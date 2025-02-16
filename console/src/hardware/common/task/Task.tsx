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

import { NULL_CLIENT_ERROR } from "@/errors";
import { Layout } from "@/layout";

export interface LayoutArgs {
  deviceKey?: device.Key;
  taskKey?: task.Key;
}

export interface Layout extends Layout.BaseState<LayoutArgs> {}

export const LAYOUT: Omit<Layout, "type"> = {
  name: "Configure",
  icon: "Task",
  location: "mosaic",
  args: {},
};

export type TaskProps<
  Config extends UnknownRecord = UnknownRecord,
  Details extends {} = UnknownRecord,
  Type extends string = string,
> =
  | {
      layoutKey: string;
      configured: false;
      task: task.Payload<Config, Details, Type>;
    }
  | {
      layoutKey: string;
      configured: true;
      task: task.Task<Config, Details, Type>;
    };

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
  Wrapped: FC<TaskProps<Config, Details, Type>>,
  options: WrapOptions<Config, Details, Type>,
): Layout.Renderer => {
  const Wrapper: Layout.Renderer = ({ layoutKey }) => {
    const { deviceKey, taskKey } = Layout.useSelectArgs<LayoutArgs>(layoutKey);
    const client = Synnax.use();
    const { data, error, isError, isPending } = useQuery<
      TaskProps<Config, Details, Type>
    >({
      queryFn: async () => {
        const { configSchema, getInitialPayload } = options;
        if (taskKey == null)
          return { configured: false, task: getInitialPayload(deviceKey), layoutKey };
        if (client == null) throw NULL_CLIENT_ERROR;
        const task = await client.hardware.tasks.retrieve<Config, Details, Type>(
          taskKey,
          { includeState: true },
        );
        task.config = configSchema.parse(task.config);
        return { configured: true, task, layoutKey };
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
      <Wrapped {...data} />
    );
    return <Eraser.Eraser>{content}</Eraser.Eraser>;
  };
  Wrapper.displayName = `TaskWrapper(${Wrapped.displayName ?? Wrapped.name})`;
  return Wrapper;
};
