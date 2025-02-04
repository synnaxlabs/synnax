// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type task } from "@synnaxlabs/client";
import { type deep } from "@synnaxlabs/x";
import { v4 as uuid } from "uuid";

import { type Layout } from "@/layout";

export type LayoutArgs<P extends task.Payload = task.Payload> = {
  create: boolean;
  initialValues?: deep.Partial<P>;
};

export const createLayoutCreator =
  <P extends task.Payload>(
    type: string,
    defaultName: string = "New Task",
    icon: string = "Task",
  ): ((args: LayoutArgs<P>) => Layout.State<LayoutArgs<P>>) =>
  (args) => {
    const key = args?.initialValues?.key ?? uuid();
    return {
      name: args?.initialValues?.name ?? defaultName,
      key,
      windowKey: key,
      icon,
      location: "mosaic",
      args,
      type,
    };
  };
