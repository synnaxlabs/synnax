// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  ConfigureAnalogRead,
  configureAnalogReadLayout,
} from "@/hardware/ni/task/AnalogRead";
import {
  ConfigureDigitalRead,
  configureDigitalReadLayout,
} from "@/hardware/ni/task/ConfigureDigitalRead";
import {
  ConfigureDigitalWrite,
  configureDigitalWriteLayout,
} from "@/hardware/ni/task/ConfigureDigitalWrite";
import { Layout } from "@/layout";

export * from "@/hardware/ni/task/AnalogRead";
export * from "@/hardware/ni/task/ConfigureDigitalRead";
export * from "@/hardware/ni/task/ConfigureDigitalWrite";
export * from "@/hardware/ni/task/types";

export const LAYOUTS: Record<string, Layout.Renderer> = {
  [configureAnalogReadLayout.type]: ConfigureAnalogRead,
  [configureDigitalWriteLayout.type]: ConfigureDigitalWrite,
  [configureDigitalReadLayout.type]: ConfigureDigitalRead,
};
