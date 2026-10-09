// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Dialog } from "@synnaxlabs/lyra/dialog";

import { MAKE } from "@/feature/kafka/device/types";
import { useConnectModal } from "@/feature/kafka/device/useConnectModal";
import { Device } from "@/platform/device";
import { Empty } from "@/platform/empty";

const EmptyContent = () => {
  const connect = useConnectModal();
  const { close: closeDialog } = Dialog.useContext();
  return (
    <Empty.Action
      message="No Kafka clusters connected"
      action="Connect cluster"
      onClick={() => {
        connect();
        closeDialog();
      }}
    />
  );
};

export const Select = () => {
  const connect = useConnectModal();
  return (
    <Device.Select
      onConfigure={(deviceKey) => connect({ deviceKey })}
      emptyContent={<EmptyContent />}
      label="Kafka cluster"
      make={MAKE}
    />
  );
};
