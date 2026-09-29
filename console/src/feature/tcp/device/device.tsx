// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { tcp } from "@synnaxlabs/client";
import { Flex } from "@synnaxlabs/lyra/flex";
import { Form } from "@synnaxlabs/lyra/form";
import { Icon } from "@synnaxlabs/lyra/icon";
import { type ReactElement } from "react";

import { Bus } from "@/feature/bus";

const HOST_INPUT_PROPS = { autoFocus: true, placeholder: "localhost" } as const;
const PORT_INPUT_PROPS = { placeholder: "5025" } as const;

const Properties = (): ReactElement => (
  <Flex.Box x>
    <Form.TextField
      path="properties.host"
      label="Host"
      grow
      inputProps={HOST_INPUT_PROPS}
    />
    <Form.NumericField
      path="properties.port"
      label="Port"
      inputProps={PORT_INPUT_PROPS}
    />
  </Flex.Box>
);

export const { MAKE, useConnectModal, Select, COMMANDS } = Bus.createDevice({
  make: "TCP",
  noun: "TCP device",
  icon: <Icon.Link />,
  properties: tcp.propertiesZ,
  integration: "tcp",
  Properties,
  getModel: () => "TCP server",
  required: { host: "Host is required" },
  getLocation: ({ host, port }) => `${host}:${port}`,
});
