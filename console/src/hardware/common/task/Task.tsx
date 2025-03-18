// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type device, type rack, task } from "@synnaxlabs/client";
import { Align, Eraser, Status, Synnax, Text, useSyncedRef } from "@synnaxlabs/pluto";
import { type UnknownRecord } from "@synnaxlabs/x";
import { useQuery } from "@tanstack/react-query";
import { type FC } from "react";
import { useStore } from "react-redux";
import { type z } from "zod";

import { NULL_CLIENT_ERROR } from "@/errors";
import { Layout } from "@/layout";
import { type RootState } from "@/store";

export interface LayoutArgs {
  deviceKey?: device.Key;
  taskKey?: task.Key;
  rackKey?: rack.Key;
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
> = {
  layoutKey: string;
  rackKey?: rack.Key;
  task: task.Payload<Config, Details, Type>;
};

export interface ConfigSchema<Config extends UnknownRecord = UnknownRecord>
  extends z.ZodType<Config, z.ZodTypeDef, unknown> {}

export interface GetInitialPayloadArgs {
  deviceKey?: device.Key;
}

export interface GetInitialPayload<
  Config extends UnknownRecord = UnknownRecord,
  Details extends {} = UnknownRecord,
  Type extends string = string,
> {
  (args: GetInitialPayloadArgs): task.Payload<Config, Details, Type>;
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
  const { configSchema, getInitialPayload } = options;
  const Wrapper: Layout.Renderer = ({ layoutKey }) => {
    const store = useStore<RootState>();
    const { deviceKey, taskKey, rackKey } = Layout.selectArgs<LayoutArgs>(
      store.getState(),
      layoutKey,
    );
    const taskKeyRef = useSyncedRef(taskKey);
    const client = Synnax.use();
    const { data, error, isError, isPending } = useQuery<
      TaskProps<Config, Details, Type>
    >({
      queryFn: async () => {
        if (taskKeyRef.current == null)
          return {
            configured: false,
            task: getInitialPayload({ deviceKey }),
            layoutKey,
            rackKey,
          };
        if (client == null) throw NULL_CLIENT_ERROR;
        const tsk = await client.hardware.tasks.retrieve<Config, Details, Type>(
          taskKeyRef.current,
          { includeState: true },
        );
        tsk.config = configSchema.parse(tsk.config);
        return {
          configured: true,
          task: tsk,
          layoutKey,
          rackKey: task.getRackKey(tsk.key),
        };
      },
      queryKey: [deviceKey, client?.key, layoutKey],
    });
    const content = isPending ? (
      <Status.Text.Centered level="h4" variant="loading">
        Fetching task from server
      </Status.Text.Centered>
    ) : isError ? (
      <Align.Space align="center" grow justify="center">
        <Text.Text color={Status.variantColors.error} level="h2">
          Failed to load data for task with key {taskKey}
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
