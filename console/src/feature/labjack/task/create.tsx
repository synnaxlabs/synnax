// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Icon } from "@synnaxlabs/lyra/icon";
import { id } from "@synnaxlabs/x";

import { MAKE } from "@/feature/labjack/device/types";
import {
  createReadChannel,
  createWriteChannel,
  READ_SCHEMAS,
  READ_TYPE,
  type ReadChannel,
  type ReadSchemas,
  WRITE_SCHEMAS,
  WRITE_TYPE,
  type WriteChannel,
  type WriteSchemas,
} from "@/feature/labjack/task/types";
import { type Selector } from "@/platform/selector";
import { Task } from "@/platform/task";

const createFirstReadChannel = (): ReadChannel => ({
  ...createReadChannel("analog"),
  key: id.create(),
});

const createFirstWriteChannel = (): WriteChannel => ({
  ...createWriteChannel("digital"),
  key: id.create(),
});

const getReadInitialValues: Task.GetInitialValues<ReadSchemas> = ({
  deviceKey,
  config,
}) => {
  const cfg = READ_SCHEMAS.config.parse(config ?? {});
  if (deviceKey != null) cfg.device = deviceKey;
  if (config == null) cfg.channels = [createFirstReadChannel()];
  return { name: "LabJack read task", type: READ_TYPE, config: cfg };
};

export const useCreateRead = Task.createUseCreate({
  getInitialValues: getReadInitialValues,
  deviceMake: MAKE,
});

export const ReadSelectable: Selector.Selectable = {
  type: READ_TYPE,
  title: "LabJack read task",
  label: "Read",
  icon: <Icon.Logo.LabJack />,
  useOnSelect: useCreateRead,
};

const getWriteInitialValues: Task.GetInitialValues<WriteSchemas> = ({
  deviceKey,
  config,
}) => {
  const cfg = WRITE_SCHEMAS.config.parse(config ?? {});
  if (deviceKey != null) cfg.device = deviceKey;
  if (config == null) cfg.channels = [createFirstWriteChannel()];
  return { name: "LabJack write task", type: WRITE_TYPE, config: cfg };
};

export const useCreateWrite = Task.createUseCreate({
  getInitialValues: getWriteInitialValues,
  deviceMake: MAKE,
});

export const WriteSelectable: Selector.Selectable = {
  type: WRITE_TYPE,
  title: "LabJack write task",
  label: "Write",
  icon: <Icon.Logo.LabJack />,
  useOnSelect: useCreateWrite,
};
