// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { can } from "@synnaxlabs/client";
import { Flex } from "@synnaxlabs/lyra/flex";
import { Form } from "@synnaxlabs/lyra/form";
import { Icon } from "@synnaxlabs/lyra/icon";
import { type ReactElement } from "react";

import { Bus } from "@/feature/bus";
import { Task } from "@/platform/task";

const BACKEND_NAMES = {
  socketcan: "SocketCAN",
  pcan: "PCAN-Basic",
  gs_usb: "gs_usb",
  slcan: "SLCAN",
  canlib: "Kvaser CANlib",
  nixnet: "NI-XNET",
} as const satisfies Record<can.Backend, string>;

const BackendField = Form.buildSelectField<can.Backend>({
  fieldKey: "backend",
  fieldProps: { label: "Backend" },
  inputProps: { resourceName: "backend", children: Task.selectItems(BACKEND_NAMES) },
});

const CHANNEL_INPUT_PROPS = { placeholder: "can0" } as const;
const BITRATE_INPUT_PROPS = { endContent: "bit/s" } as const;

const Properties = (): ReactElement => {
  const fd = Form.useFieldValue<boolean>("properties.fd");
  return (
    <>
      <Flex.Box x>
        <BackendField path="properties" />
        <Form.TextField
          path="properties.channel"
          label="Channel"
          grow
          inputProps={CHANNEL_INPUT_PROPS}
        />
      </Flex.Box>
      <Flex.Box x>
        <Form.NumericField
          path="properties.bitrate"
          label="Bitrate"
          inputProps={BITRATE_INPUT_PROPS}
        />
        {fd && (
          <Form.NumericField
            path="properties.dataBitrate"
            label="Data bitrate"
            inputProps={BITRATE_INPUT_PROPS}
          />
        )}
      </Flex.Box>
      <Flex.Box x>
        <Form.SwitchField path="properties.fd" label="CAN FD" />
        <Form.SwitchField path="properties.listenOnly" label="Listen only" />
      </Flex.Box>
    </>
  );
};

export const { MAKE, useConnectModal, Select, COMMANDS } = Bus.createDevice({
  make: "CAN",
  noun: "CAN device",
  icon: <Icon.Hardware />,
  properties: can.propertiesZ,
  integration: "can",
  Properties,
  getModel: ({ backend }) => backend,
  required: { channel: "Channel is required" },
  getLocation: ({ channel }) => channel,
});
