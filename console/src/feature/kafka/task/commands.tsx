// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { task } from "@synnaxlabs/client";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Access } from "@synnaxlabs/pluto";

import { useCreateRead } from "@/feature/kafka/task/Read";
import { useCreateWrite } from "@/feature/kafka/task/Write";
import { FLAGS } from "@/flags";
import { Command } from "@/platform/command";

const useVisible = () => {
  const canCreate = Access.useCreateGranted(task.TYPE_ONTOLOGY_ID);
  return FLAGS.kafka && canCreate;
};

const CreateReadCommand = Command.create({
  key: "kafka_create_read_task",
  name: "Create Kafka read task",
  icon: <Icon.Logo.Kafka />,
  useOnSelect: useCreateRead,
  useVisible,
});

const CreateWriteCommand = Command.create({
  key: "kafka_create_write_task",
  name: "Create Kafka write task",
  icon: <Icon.Logo.Kafka />,
  useOnSelect: useCreateWrite,
  useVisible,
});

export const COMMANDS = [CreateReadCommand, CreateWriteCommand];
