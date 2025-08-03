// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Flex, Icon, Text } from "@synnaxlabs/pluto";

import { Common } from "@/hardware/common";
import { CONNECT_LAYOUT } from "@/hardware/opc/device/Connect";
import { MAKE } from "@/hardware/opc/device/types";
import { Layout } from "@/layout";

const EmptyContent = () => {
  const placeLayout = Layout.usePlacer();
  return (
    <Flex.Box>
      <Text.Text color={10}>No OPC UA servers connected.</Text.Text>
      <Text.Text onClick={() => placeLayout(CONNECT_LAYOUT)}>
        Connect a new server.
      </Text.Text>
    </Flex.Box>
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
