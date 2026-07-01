// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Dialog, Icon } from "@synnaxlabs/pluto";

import { Device as CommonDevice } from "@/component/device";
import { Empty } from "@/component/empty";
import { MAKE } from "@/service/opc/device/types";
import { useConnectModal } from "@/service/opc/device/useConnectModal";

const EmptyContent = () => {
  const connect = useConnectModal();
  const { close } = Dialog.useContext();
  return (
    <Empty.Action
      message="No OPC UA servers connected."
      action="Connect a new server"
      full="y"
      onClick={() => {
        connect();
        close();
      }}
    />
  );
};

export const Select = () => {
  const connect = useConnectModal();
  return (
    <CommonDevice.Select
      onConfigure={(deviceKey) => connect({ deviceKey })}
      emptyContent={<EmptyContent />}
      label="OPC UA Server"
      make={MAKE}
      icon={<Icon.Logo.OPC />}
    />
  );
};
