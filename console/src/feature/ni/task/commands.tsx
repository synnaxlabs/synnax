// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Icon } from "@synnaxlabs/pluto";

import { useCreateAnalogRead } from "@/feature/ni/task/AnalogRead";
import { useCreateAnalogWrite } from "@/feature/ni/task/AnalogWrite";
import { useCreateCounterRead } from "@/feature/ni/task/CounterRead";
import { useCreateDigitalRead } from "@/feature/ni/task/DigitalRead";
import { useCreateDigitalWrite } from "@/feature/ni/task/DigitalWrite";
import { Task } from "@/platform/task";

const CreateAnalogReadCommand = Task.createCommand({
  key: "ni_create_analog_read_task",
  name: "Create an NI Analog Read Task",
  icon: <Icon.Logo.NI />,
  useOnSelect: useCreateAnalogRead,
});

const CreateAnalogWriteCommand = Task.createCommand({
  key: "ni_create_analog_write_task",
  name: "Create an NI Analog Write Task",
  icon: <Icon.Logo.NI />,
  useOnSelect: useCreateAnalogWrite,
});

const CreateCounterReadCommand = Task.createCommand({
  key: "ni_create_counter_read_task",
  name: "Create an NI Counter Read Task",
  icon: <Icon.Logo.NI />,
  useOnSelect: useCreateCounterRead,
});

const CreateDigitalWriteCommand = Task.createCommand({
  key: "ni_create_digital_write_task",
  name: "Create an NI Digital Write Task",
  icon: <Icon.Logo.NI />,
  useOnSelect: useCreateDigitalWrite,
});

const CreateDigitalReadCommand = Task.createCommand({
  key: "ni_create_digital_read_task",
  name: "Create an NI Digital Read Task",
  icon: <Icon.Logo.NI />,
  useOnSelect: useCreateDigitalRead,
});

export const COMMANDS = [
  CreateAnalogReadCommand,
  CreateAnalogWriteCommand,
  CreateCounterReadCommand,
  CreateDigitalWriteCommand,
  CreateDigitalReadCommand,
];
