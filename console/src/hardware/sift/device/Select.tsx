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
import { CONNECT_LAYOUT } from "@/hardware/sift/device/Connect";
import { MAKE } from "@/hardware/sift/device/types";
import { Layout } from "@/layout";

const EmptyContent = () => {
  const placeLayout = Layout.usePlacer();
  const { close } = Dialog.useContext();
  return (
    <EmptyAction
      message="No Sift connections."
      action="Connect to Sift"
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
    label="Sift Server"
    path="deviceKey"
    make={MAKE}
    icon={<Icon.Logo.Sift />}
  />
);
