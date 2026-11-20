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

import { MAKE } from "@/feature/modbus/device/types";
import {
  nextOpenChannel,
  READ_CHANNEL_SCHEMAS,
  READ_SCHEMAS,
  READ_TYPE,
  type ReadChannel,
  type ReadSchemas,
  readsOverlap,
  WRITE_CHANNEL_SCHEMAS,
  WRITE_SCHEMAS,
  WRITE_TYPE,
  type WriteChannel,
  type WriteSchemas,
  writesOverlap,
} from "@/feature/modbus/task/types";
import type { Selector } from "@/platform/selector";
import { Task } from "@/platform/task";

export const getOpenReadChannel = (
  channels: ReadChannel[],
  copy?: string,
): ReadChannel => {
  const next = nextOpenChannel(channels, readsOverlap, copy);
  if (next == null)
    return {
      ...READ_CHANNEL_SCHEMAS.coil.parse({ type: "coil" }),
      key: id.create(),
    };
  return {
    ...next,
    ...Task.READ_CHANNEL_OVERRIDE,
    key: id.create(),
  };
};

export const getOpenWriteChannel = (
  channels: WriteChannel[],
  copy?: string,
): WriteChannel => {
  const next = nextOpenChannel(channels, writesOverlap, copy);
  if (next == null)
    return {
      ...WRITE_CHANNEL_SCHEMAS.coil.parse({ type: "coil" }),
      key: id.create(),
    };
  return {
    ...next,
    ...Task.READ_CHANNEL_OVERRIDE,
    key: id.create(),
  };
};

const getReadInitialValues: Task.GetInitialValues<ReadSchemas> = ({ deviceKey }) => {
  const config = READ_SCHEMAS.config.parse({});
  if (deviceKey != null) config.device = deviceKey;
  config.channels = [getOpenReadChannel([])];
  return { name: "Modbus read task", type: READ_TYPE, config };
};

export const useCreateRead = Task.createUseCreate({
  getInitialValues: getReadInitialValues,
  deviceMake: MAKE,
});

export const ReadSelectable: Selector.Selectable = {
  type: READ_TYPE,
  title: "Modbus read task",
  label: "Read",
  icon: <Icon.Logo.Modbus />,
  useOnSelect: useCreateRead,
};

const getWriteInitialValues: Task.GetInitialValues<WriteSchemas> = ({ deviceKey }) => {
  const config = WRITE_SCHEMAS.config.parse({});
  if (deviceKey != null) config.device = deviceKey;
  config.channels = [getOpenWriteChannel([])];
  return { name: "Modbus write task", type: WRITE_TYPE, config };
};

export const useCreateWrite = Task.createUseCreate({
  getInitialValues: getWriteInitialValues,
  deviceMake: MAKE,
});

export const WriteSelectable: Selector.Selectable = {
  type: WRITE_TYPE,
  title: "Modbus write task",
  label: "Write",
  icon: <Icon.Logo.Modbus />,
  useOnSelect: useCreateWrite,
};
