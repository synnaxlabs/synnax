// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { device } from "@synnaxlabs/client";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Access } from "@synnaxlabs/pluto";

import { useConnectModal } from "@/feature/kafka/device/useConnectModal";
import { FLAGS } from "@/flags";
import { Command } from "@/platform/command";

const useVisible = () => {
  const canCreate = Access.useCreateGranted(device.TYPE_ONTOLOGY_ID);
  return FLAGS.kafka && canCreate;
};

const ConnectClusterCommand = Command.create({
  key: "kafka_connect_cluster",
  name: "Connect Kafka cluster",
  icon: <Icon.Logo.Kafka />,
  useOnSelect: useConnectModal,
  useVisible,
});

export const COMMANDS = [ConnectClusterCommand];
