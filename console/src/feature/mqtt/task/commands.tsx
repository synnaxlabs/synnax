// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Icon } from "@synnaxlabs/pluto";

import { useCreateEdge } from "@/feature/mqtt/task/Edge";
import { useCreateRead } from "@/feature/mqtt/task/Read";
import { useCreateWrite } from "@/feature/mqtt/task/Write";
import { Task } from "@/platform/task";

const CreateReadCommand = Task.createCommand({
  key: "mqtt_create_read_task",
  name: "Create MQTT read task",
  icon: <Icon.Logo.MQTT />,
  useOnSelect: useCreateRead,
});

const CreateWriteCommand = Task.createCommand({
  key: "mqtt_create_write_task",
  name: "Create MQTT write task",
  icon: <Icon.Logo.MQTT />,
  useOnSelect: useCreateWrite,
});

const CreateEdgeCommand = Task.createCommand({
  key: "mqtt_create_sparkplug_edge_task",
  name: "Create Sparkplug edge node",
  icon: <Icon.Logo.MQTT />,
  useOnSelect: useCreateEdge,
});

export const COMMANDS = [CreateReadCommand, CreateWriteCommand, CreateEdgeCommand];
