// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Device } from "@/hardware/ni/device";
import { Task } from "@/hardware/ni/task";
import { type Layout } from "@/layout";

export * from "@/hardware/ni/device";
export * from "@/hardware/ni/palette";
export * from "@/hardware/ni/task";

export const LAYOUTS: Record<string, Layout.Renderer> = {
  ...Task.LAYOUTS,
  ...Device.LAYOUTS,
};

export const SELECTABLES = Task.SELECTABLES;

export const MAKE = "NI";
export type Make = typeof MAKE;
