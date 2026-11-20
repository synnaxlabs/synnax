// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ethercat } from "@synnaxlabs/client";
import { Icon } from "@synnaxlabs/lyra/icon";

import { type SlaveDevice } from "@/feature/ethercat/device/types";
import { createSlaveChannels } from "@/feature/ethercat/task/configure";
import {
  createEntryReadChannel,
  createEntryWriteChannel,
  createReadChannel,
  createWriteChannel,
  type EntryReadChannel,
  isCalibrated,
  READ_SCHEMAS,
  READ_TYPE,
  type ReadSchemas,
  WRITE_SCHEMAS,
  WRITE_TYPE,
  type WriteSchemas,
} from "@/feature/ethercat/task/types";
import { type Selector } from "@/platform/selector";
import { Task } from "@/platform/task";

/** @returns a channel on the entry, calibrated when the Driver calibrates the entry. */
export const createSlaveReadChannel = (
  { key, properties }: SlaveDevice,
  entry: ethercat.PDOEntry,
): EntryReadChannel => ({
  ...createEntryReadChannel(key, entry),
  calibrated: isCalibrated(properties, entry),
});

export const createSlaveWriteChannel = (
  { key }: SlaveDevice,
  entry: ethercat.PDOEntry,
) => createEntryWriteChannel(key, entry);

const getReadInitialValues: Task.GetInitialValues<ReadSchemas> = async ({
  client,
  deviceKeys,
  config,
}) => {
  const cfg = READ_SCHEMAS.config.parse(config ?? {});
  if (config == null)
    cfg.channels =
      deviceKeys.length === 0
        ? [createReadChannel([])]
        : await createSlaveChannels(
            client,
            deviceKeys,
            "inputs",
            createSlaveReadChannel,
            createReadChannel([]),
          );
  return { name: "EtherCAT read task", type: READ_TYPE, config: cfg };
};

export const useCreateRead = Task.createUseCreate({
  getInitialValues: getReadInitialValues,
});

export const ReadSelectable: Selector.Selectable = {
  type: READ_TYPE,
  title: "EtherCAT read task",
  label: "Read",
  icon: <Icon.Logo.EtherCAT />,
  useOnSelect: useCreateRead,
};

const getWriteInitialValues: Task.GetInitialValues<WriteSchemas> = async ({
  client,
  deviceKeys,
  config,
}) => {
  const cfg = WRITE_SCHEMAS.config.parse(config ?? {});
  if (config == null)
    cfg.channels =
      deviceKeys.length === 0
        ? [createWriteChannel([])]
        : await createSlaveChannels(
            client,
            deviceKeys,
            "outputs",
            createSlaveWriteChannel,
            createWriteChannel([]),
          );
  return { name: "EtherCAT write task", type: WRITE_TYPE, config: cfg };
};

export const useCreateWrite = Task.createUseCreate({
  getInitialValues: getWriteInitialValues,
});

export const WriteSelectable: Selector.Selectable = {
  type: WRITE_TYPE,
  title: "EtherCAT write task",
  label: "Write",
  icon: <Icon.Logo.EtherCAT />,
  useOnSelect: useCreateWrite,
};
