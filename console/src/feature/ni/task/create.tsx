// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Icon } from "@synnaxlabs/lyra/icon";

import { MAKE } from "@/feature/ni/device/types";
import {
  createNextAIChannel,
  createNextAOChannel,
  createNextCIChannel,
  createNextDIChannel,
  createNextDOChannel,
} from "@/feature/ni/task/createChannel";
import {
  ANALOG_READ_TYPE,
  ANALOG_WRITE_TYPE,
  analogReadConfigZ,
  type AnalogReadSchemas,
  analogWriteConfigZ,
  type AnalogWriteSchemas,
  COUNTER_READ_TYPE,
  counterReadConfigZ,
  type CounterReadSchemas,
  DIGITAL_READ_TYPE,
  DIGITAL_WRITE_TYPE,
  digitalReadConfigZ,
  type DigitalReadSchemas,
  digitalWriteConfigZ,
  type DigitalWriteSchemas,
} from "@/feature/ni/task/types";
import type { Selector } from "@/platform/selector";
import { Task } from "@/platform/task";

const getAnalogReadInitialValues: Task.GetInitialValues<AnalogReadSchemas> = ({
  deviceKeys,
  config,
}) => {
  const cfg = analogReadConfigZ.parse(config ?? {});
  if (config == null) {
    cfg.channels = [];
    if (deviceKeys.length === 0) cfg.channels.push(createNextAIChannel([]));
    for (const device of deviceKeys)
      cfg.channels.push(createNextAIChannel(cfg.channels, { device }));
  }
  return { name: "NI analog read task", type: ANALOG_READ_TYPE, config: cfg };
};

export const useCreateAnalogRead = Task.createUseCreate({
  getInitialValues: getAnalogReadInitialValues,
  deviceMake: MAKE,
});

export const AnalogReadSelectable: Selector.Selectable = {
  type: ANALOG_READ_TYPE,
  title: "NI analog read task",
  label: "Analog read",
  icon: <Icon.Logo.NI />,
  useOnSelect: useCreateAnalogRead,
};

const getAnalogWriteInitialValues: Task.GetInitialValues<AnalogWriteSchemas> = ({
  deviceKey,
  config,
}) => {
  const cfg = analogWriteConfigZ.parse(config ?? {});
  if (deviceKey != null) cfg.device = deviceKey;
  if (config == null) cfg.channels = [createNextAOChannel([])];
  return { name: "NI analog write task", type: ANALOG_WRITE_TYPE, config: cfg };
};

export const useCreateAnalogWrite = Task.createUseCreate({
  getInitialValues: getAnalogWriteInitialValues,
  deviceMake: MAKE,
});

export const AnalogWriteSelectable: Selector.Selectable = {
  type: ANALOG_WRITE_TYPE,
  title: "NI analog write task",
  label: "Analog write",
  icon: <Icon.Logo.NI />,
  useOnSelect: useCreateAnalogWrite,
};

const getCounterReadInitialValues: Task.GetInitialValues<CounterReadSchemas> = ({
  deviceKeys,
  config,
}) => {
  const cfg = counterReadConfigZ.parse(config ?? {});
  if (config == null) {
    cfg.channels = [];
    if (deviceKeys.length === 0) cfg.channels.push(createNextCIChannel([]));
    for (const device of deviceKeys)
      cfg.channels.push(createNextCIChannel(cfg.channels, { device }));
  }
  return { name: "NI counter read task", type: COUNTER_READ_TYPE, config: cfg };
};

export const useCreateCounterRead = Task.createUseCreate({
  getInitialValues: getCounterReadInitialValues,
  deviceMake: MAKE,
});

export const CounterReadSelectable: Selector.Selectable = {
  type: COUNTER_READ_TYPE,
  title: "NI counter read task",
  label: "Counter read",
  icon: <Icon.Logo.NI />,
  useOnSelect: useCreateCounterRead,
};

const getDigitalReadInitialValues: Task.GetInitialValues<DigitalReadSchemas> = ({
  deviceKey,
  config,
}) => {
  const cfg = digitalReadConfigZ.parse(config ?? {});
  if (deviceKey != null) cfg.device = deviceKey;
  if (config == null) cfg.channels = [createNextDIChannel([])];
  return { name: "NI digital read task", type: DIGITAL_READ_TYPE, config: cfg };
};

export const useCreateDigitalRead = Task.createUseCreate({
  getInitialValues: getDigitalReadInitialValues,
  deviceMake: MAKE,
});

export const DigitalReadSelectable: Selector.Selectable = {
  type: DIGITAL_READ_TYPE,
  title: "NI digital read task",
  label: "Digital read",
  icon: <Icon.Logo.NI />,
  useOnSelect: useCreateDigitalRead,
};

const getDigitalWriteInitialValues: Task.GetInitialValues<DigitalWriteSchemas> = ({
  deviceKey,
  config,
}) => {
  const cfg = digitalWriteConfigZ.parse(config ?? {});
  if (deviceKey != null) cfg.device = deviceKey;
  if (config == null) cfg.channels = [createNextDOChannel([])];
  return { name: "NI digital write task", type: DIGITAL_WRITE_TYPE, config: cfg };
};

export const useCreateDigitalWrite = Task.createUseCreate({
  getInitialValues: getDigitalWriteInitialValues,
  deviceMake: MAKE,
});

export const DigitalWriteSelectable: Selector.Selectable = {
  type: DIGITAL_WRITE_TYPE,
  title: "NI digital write task",
  label: "Digital write",
  icon: <Icon.Logo.NI />,
  useOnSelect: useCreateDigitalWrite,
};
