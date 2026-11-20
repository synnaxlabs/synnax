// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Icon } from "@synnaxlabs/lyra/icon";

import { MAKE } from "@/feature/opcua/device/types";
import {
  READ_SCHEMAS,
  READ_TYPE,
  type ReadSchemas,
  WRITE_SCHEMAS,
  WRITE_TYPE,
  type WriteSchemas,
} from "@/feature/opcua/task/types";
import { type Selector } from "@/platform/selector";
import { Task } from "@/platform/task";

const getReadInitialValues: Task.GetInitialValues<ReadSchemas> = ({
  deviceKey,
  config,
}) => {
  const cfg = READ_SCHEMAS.config.parse(config ?? {});
  if (deviceKey != null) cfg.device = deviceKey;
  return { name: "OPC UA read task", type: READ_TYPE, config: cfg };
};

export const useCreateRead = Task.createUseCreate({
  getInitialValues: getReadInitialValues,
  deviceMake: MAKE,
});

export const ReadSelectable: Selector.Selectable = {
  type: READ_TYPE,
  title: "OPC UA read task",
  label: "Read",
  icon: <Icon.Logo.OPCUA />,
  useOnSelect: useCreateRead,
};

const getWriteInitialValues: Task.GetInitialValues<WriteSchemas> = ({
  deviceKey,
  config,
}) => {
  const cfg = WRITE_SCHEMAS.config.parse(config ?? {});
  if (deviceKey != null) cfg.device = deviceKey;
  return { name: "OPC UA write task", type: WRITE_TYPE, config: cfg };
};

export const useCreateWrite = Task.createUseCreate({
  getInitialValues: getWriteInitialValues,
  deviceMake: MAKE,
});

export const WriteSelectable: Selector.Selectable = {
  type: WRITE_TYPE,
  title: "OPC UA write task",
  label: "Write",
  icon: <Icon.Logo.OPCUA />,
  useOnSelect: useCreateWrite,
};
