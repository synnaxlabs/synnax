// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type device, type rack, task } from "@synnaxlabs/client";
import { Task } from "@synnaxlabs/pluto";
import { type record } from "@synnaxlabs/x";
import { type FC } from "react";
import { useStore } from "react-redux";
import { type z } from "zod/v4";

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
  Config extends record.Unknown = record.Unknown,
  Details extends {} = record.Unknown,
  Type extends string = string,
> = {
  layoutKey: string;
  rackKey?: rack.Key;
  task: task.Payload<Config, Details, Type>;
};

export interface ConfigSchema<Config extends record.Unknown = record.Unknown>
  extends z.ZodType<Config> {}

export interface GetInitialPayloadArgs {
  deviceKey?: device.Key;
}

export interface GetInitialPayload<
  Config extends record.Unknown = record.Unknown,
  Details extends {} = record.Unknown,
  Type extends string = string,
> {
  (args: GetInitialPayloadArgs): task.Payload<Config, Details, Type>;
}

export interface WrapOptions<
  Config extends record.Unknown = record.Unknown,
  Details extends {} = record.Unknown,
  Type extends string = string,
> {
  configSchema: ConfigSchema<Config>;
  getInitialPayload: GetInitialPayload<Config, Details, Type>;
}

export const wrap = <
  Config extends record.Unknown = record.Unknown,
  Details extends {} = record.Unknown,
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
    const res = Task.use(taskKey);
    if (res.status !== "success") return res.Status;
    const data: TaskProps<Config, Details, Type> = {
      task: getInitialPayload({ deviceKey }),
      layoutKey,
      rackKey,
    };
    if (res.data != null) {
      const tsk = res.data;
      try {
        data.task.config = configSchema.parse(tsk.config);
      } catch (e) {
        console.error(`Failed to parse config for ${tsk.name}`, tsk.config, e);
        throw e;
      }
      data.task = tsk as unknown as task.Payload<Config, Details, Type>;
      data.rackKey = task.getRackKey(tsk.key);
    }
    return <Wrapped {...data} />;
  };
  Wrapper.displayName = `TaskWrapper(${Wrapped.displayName ?? Wrapped.name})`;
  return Wrapper;
};
