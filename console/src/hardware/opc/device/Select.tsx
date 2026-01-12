// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Dialog, Icon } from "@synnaxlabs/pluto";

import { EmptyAction } from "@/components";
import { Common } from "@/hardware/common";
import { CONNECT_LAYOUT } from "@/hardware/opc/device/Connect";
import { MAKE } from "@/hardware/opc/device/types";
import { Layout } from "@/layout";

const EmptyContent = () => {
  const placeLayout = Layout.usePlacer();
  const { close } = Dialog.useContext();
  return (
    <EmptyAction
      message="No OPC UA servers connected."
      action="Connect a new server"
      full="y"
      onClick={() => {
        placeLayout(CONNECT_LAYOUT);
        close();
      }}
    />
  );
};

export const Select = () => (
  <Common.Device.Select
    configureLayout={CONNECT_LAYOUT}
    emptyContent={<EmptyContent />}
    label="OPC UA Server"
    make={MAKE}
    icon={<Icon.Logo.OPC />}
  />
);
