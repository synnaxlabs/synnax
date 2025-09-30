// Copyright 2025 Synnax Labs, Inc.
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
import { CONNECT_LAYOUT } from "@/hardware/modbus/device/Connect";
import { MAKE } from "@/hardware/modbus/device/types";
import { Layout } from "@/layout";

const EmptyContent = () => {
  const placeLayout = Layout.usePlacer();
  const { close: closeDialog } = Dialog.useContext();
  return (
    <EmptyAction
      message="No Modbus servers connected."
      action="Connect a new server"
      onClick={() => {
        placeLayout(CONNECT_LAYOUT);
        closeDialog();
      }}
    />
  );
};

export const Select = () => (
  <Common.Device.Select
    configureLayout={CONNECT_LAYOUT}
    emptyContent={<EmptyContent />}
    label="Modbus Server"
    make={MAKE}
  />
);
