// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type device, type rack, task } from "@synnaxlabs/client";
import { Status, Task } from "@synnaxlabs/pluto";
import { type FC } from "react";
import { useStore } from "react-redux";
import { type z } from "zod";

import { Layout } from "@/layout";
import { type RootState } from "@/store";

export interface LayoutArgs {
  deviceKey?: device.Key;
  taskKey?: task.Key;
  rackKey?: rack.Key;
  config?: unknown;
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
  config?: unknown;
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
  const useRetrieve = Task.createRetrieveQuery(schemas).useDirect;
  const Wrapper: Layout.Renderer = ({ layoutKey }) => {
    const store = useStore<RootState>();
    const { deviceKey, taskKey, rackKey, config } = Layout.selectArgs<LayoutArgs>(
      store.getState(),
      layoutKey,
    );
    const { data, variant, status } = useRetrieve({
      params: { key: taskKey },
    });
    if (variant !== "success")
      return (
        <Status.Summary
          variant={variant}
          message={status.message}
          description={status.description}
          center
        />
      );
    return (
      <Wrapped
        rackKey={data ? task.rackKey(data.key) : rackKey}
        task={data ?? getInitialPayload({ deviceKey, config })}
        layoutKey={layoutKey}
      />
    );
  };
  Wrapper.displayName = `TaskWrapper(${Wrapped.displayName ?? Wrapped.name})`;
  return Wrapper;
};
