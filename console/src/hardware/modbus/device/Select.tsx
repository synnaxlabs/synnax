// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Dialog } from "@synnaxlabs/pluto";

import { EmptyAction } from "@/components";
import { Common } from "@/hardware/common";
import { useConnect } from "@/hardware/modbus/device/Connect";
import { MAKE } from "@/hardware/modbus/device/types";

const EmptyContent = () => {
  const connect = useConnect();
  const { close: closeDialog } = Dialog.useContext();
  return (
    <EmptyAction
      message="No Modbus servers connected."
      action="Connect a new server"
      onClick={() => {
        connect({});
        closeDialog();
      }}
    />
  );
};

export const Select = () => {
  const connect = useConnect();
  return (
    <Common.Device.Select
      onConfigure={(deviceKey) => connect({ deviceKey })}
      emptyContent={<EmptyContent />}
      label="Modbus Server"
      make={MAKE}
    />
  );
};
