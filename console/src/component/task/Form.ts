// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type device, type rack, type Synnax, type task } from "@synnaxlabs/client";
import { type Flux, Form as PForm, type Task } from "@synnaxlabs/pluto";
import { type FC } from "react";
import { type z } from "zod";

import { useStatus } from "@/component/task/useStatus";

export interface OnConfigure<Config extends z.ZodType = z.ZodType> {
  (
    client: Synnax,
    config: z.infer<Config>,
    name: string,
  ): Promise<[z.infer<Config>, rack.Key]>;
}

export interface FormLayoutArgs {
  deviceKey?: device.Key;
  taskKey?: task.Key;
  rackKey?: rack.Key;
  config?: unknown;
}

export interface getInitialValuesArgs {
  deviceKey?: device.Key;
  config?: unknown;
}

export interface GetInitialValues<S extends task.Schemas = task.Schemas> {
  (args: getInitialValuesArgs): Task.InitialValues<S>;
}

export interface FormProps<
  S extends task.Schemas = task.Schemas,
> extends PForm.UseReturn<Task.FormSchema<S>> {
  layoutKey: string;
  status: Flux.Result<undefined>["status"];
  onConfigure: () => void;
}

export interface WrapFormArgs<S extends task.Schemas = task.Schemas> {
  Properties?: FC<{}>;
  Form: FC<FormProps<S>>;
  type: z.infer<S["type"]>;
  onConfigure: OnConfigure<S["config"]>;
  schemas: S;
  getInitialValues: GetInitialValues<S>;
  showHeader?: boolean;
  showControls?: boolean;
}

export const useIsRunning = <Schema extends z.ZodType>(
  ctx?: PForm.ContextValue<Schema>,
) => useStatus(ctx)?.details.running ?? false;

export const useIsSnapshot = <Schema extends z.ZodType>(
  ctx?: PForm.ContextValue<Schema>,
) => PForm.useFieldValue<boolean>("snapshot", { ctx });
