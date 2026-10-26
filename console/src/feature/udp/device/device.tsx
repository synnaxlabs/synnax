// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { udp } from "@synnaxlabs/client";
import { Flex } from "@synnaxlabs/lyra/flex";
import { Form } from "@synnaxlabs/lyra/form";
import { Icon } from "@synnaxlabs/lyra/icon";
import { type ReactElement } from "react";

import { Bus } from "@/feature/bus";

const PORT_INPUT_PROPS = { placeholder: "5000" } as const;
const REMOTE_HOST_INPUT_PROPS = { placeholder: "None" } as const;
const MULTICAST_INPUT_PROPS = { placeholder: "None" } as const;
const LOCAL_ADDRESS_INPUT_PROPS = { placeholder: "All interfaces" } as const;

const Properties = (): ReactElement => (
  <>
    <Flex.Box x>
      <Form.TextField
        path="properties.localAddress"
        label="Local address"
        grow
        inputProps={LOCAL_ADDRESS_INPUT_PROPS}
      />
      <Form.NumericField
        path="properties.port"
        label="Local port"
        inputProps={PORT_INPUT_PROPS}
      />
    </Flex.Box>
    <Flex.Box x>
      <Form.TextField
        path="properties.remoteHost"
        label="Remote host"
        grow
        inputProps={REMOTE_HOST_INPUT_PROPS}
      />
      <Form.NumericField path="properties.remotePort" label="Remote port" />
    </Flex.Box>
    <Form.TextField
      path="properties.multicastGroup"
      label="Multicast group"
      inputProps={MULTICAST_INPUT_PROPS}
    />
  </>
);

export const { MAKE, useConnectModal, Select, COMMANDS, NETWORK } = Bus.createDevice({
  make: "UDP",
  name: "UDP",
  noun: "UDP device",
  description: "Read and write ICD messages over UDP.",
  icon: <Icon.Bridge />,
  properties: udp.propertiesZ,
  integration: "udp",
  Properties,
  getModel: () => "UDP socket",
  getLocation: ({ localAddress, port }) => `${localAddress}:${port}`,
});
