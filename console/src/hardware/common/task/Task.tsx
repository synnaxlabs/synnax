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
import { useQuery } from "@tanstack/react-query";
import { type FC } from "react";
import { useStore } from "react-redux";
import { type z } from "zod/v4";

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
  Type extends z.ZodLiteral<string> = z.ZodLiteral<string>,
  Config extends z.ZodType = z.ZodType,
  StatusData extends z.ZodType = z.ZodType,
> = {
  layoutKey: string;
  rackKey?: rack.Key;
  task: task.Payload<Type, Config, StatusData>;
};

export interface GetInitialPayloadArgs {
  deviceKey?: device.Key;
}

export interface GetInitialPayload<
  Type extends z.ZodLiteral<string> = z.ZodLiteral<string>,
  Config extends z.ZodType = z.ZodType,
  StatusData extends z.ZodType = z.ZodType,
> {
  (args: GetInitialPayloadArgs): task.Payload<Type, Config, StatusData>;
}

export interface WrapOptions<
  Type extends z.ZodLiteral<string> = z.ZodLiteral<string>,
  Config extends z.ZodType = z.ZodType,
  StatusData extends z.ZodType = z.ZodType,
> {
  schemas: task.Schemas<Type, Config, StatusData>;
  getInitialPayload: GetInitialPayload<Type, Config, StatusData>;
}

export const wrap = <
  Type extends z.ZodLiteral<string> = z.ZodLiteral<string>,
  Config extends z.ZodType = z.ZodType,
  StatusData extends z.ZodType = z.ZodType,
>(
  Wrapped: FC<TaskProps<Type, Config, StatusData>>,
  options: WrapOptions<Type, Config, StatusData>,
): Layout.Renderer => {
  const { schemas, getInitialPayload } = options;
  const Wrapper: Layout.Renderer = ({ layoutKey }) => {
    const store = useStore<RootState>();
    const { deviceKey, taskKey, rackKey } = Layout.selectArgs<LayoutArgs>(
      store.getState(),
      layoutKey,
    );
    const taskKeyRef = useSyncedRef(taskKey);
    const client = Synnax.use();
    const { data, error, isError, isPending } = useQuery<
      TaskProps<Type, Config, StatusData>
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
        const tsk = await client.hardware.tasks.retrieve<Type, Config, StatusData>({
          key: taskKeyRef.current,
          includeStatus: true,
          schemas,
        });
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
        <Text.Text color={Status.VARIANT_COLORS.error} level="h2">
          Failed to load data for task with key {taskKey}
        </Text.Text>
        <Text.Text color={Status.VARIANT_COLORS.error} level="p">
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
