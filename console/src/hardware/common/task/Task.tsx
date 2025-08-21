// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type device, type rack, type task } from "@synnaxlabs/client";
import { type Task } from "@synnaxlabs/pluto";
import { type z } from "zod";

import { type Layout } from "@/layout";

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

export interface GetInitialValues<
  Type extends z.ZodLiteral<string> = z.ZodLiteral<string>,
  Config extends z.ZodType = z.ZodType,
  StatusData extends z.ZodType = z.ZodType,
> {
  (args: GetInitialPayloadArgs): Task.InitialValues<Type, Config, StatusData>;
}

export interface WrapOptions<
  Type extends z.ZodLiteral<string> = z.ZodLiteral<string>,
  Config extends z.ZodType = z.ZodType,
  StatusData extends z.ZodType = z.ZodType,
> {
  schemas: task.Schemas<Type, Config, StatusData>;
  getInitialPayload: GetInitialValues<Type, Config, StatusData>;
}
