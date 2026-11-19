// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Icon } from "@synnaxlabs/lyra/icon";

import { MAKE } from "@/feature/sift/device/types";
import {
  STREAM_SCHEMAS,
  STREAM_TYPE,
  type StreamSchemas,
} from "@/feature/sift/task/types";
import { type Selector } from "@/platform/selector";
import { Task } from "@/platform/task";

const getStreamInitialValues: Task.GetInitialValues<StreamSchemas> = ({
  deviceKey,
  config,
}) => {
  const cfg = STREAM_SCHEMAS.config.parse(config ?? {});
  if (deviceKey != null) cfg.device = deviceKey;
  return { name: "Sift stream task", type: STREAM_TYPE, config: cfg };
};

export const useCreateStream = Task.createUseCreate({
  getInitialValues: getStreamInitialValues,
  deviceMake: MAKE,
});

export const StreamSelectable: Selector.Selectable = {
  type: STREAM_TYPE,
  title: "Sift stream task",
  label: "Stream",
  icon: <Icon.Logo.Sift />,
  useOnSelect: useCreateStream,
};
