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
import { deep } from "@synnaxlabs/x";
import { type ReactElement } from "react";

import { Bus } from "@/feature/bus";
import { Task } from "@/platform/task";

const PATH = deep.path<{ properties: can.Properties }>().properties;

const BACKEND_NAMES = {
  socketcan: "SocketCAN",
  pcan: "PCAN-Basic",
  gs_usb: "gs_usb",
  slcan: "SLCAN",
  canlib: "Kvaser CANlib",
  nixnet: "NI-XNET",
} as const satisfies Record<can.Backend, string>;

const BackendField = Form.buildSelectField<can.Backend>({
  fieldProps: { label: "Backend" },
  inputProps: { resourceName: "backend", children: Task.selectItems(BACKEND_NAMES) },
});

const CHANNEL_INPUT_PROPS = { placeholder: "can0" } as const;
const BITRATE_INPUT_PROPS = { endContent: "bit/s" } as const;

const Properties = (): ReactElement => {
  const fd = Form.useFieldValue(PATH.fd);
  return (
    <>
      <Flex.Box x>
        <BackendField path={PATH.backend} />
        <Form.TextField
          path={PATH.channel}
          label="Channel"
          grow
          inputProps={CHANNEL_INPUT_PROPS}
        />
      </Flex.Box>
      <Flex.Box x>
        <Form.NumericField
          path={PATH.bitrate}
          label="Bitrate"
          inputProps={BITRATE_INPUT_PROPS}
        />
        {fd && (
          <Form.NumericField
            path={PATH.dataBitrate}
            label="Data bitrate"
            inputProps={BITRATE_INPUT_PROPS}
          />
        )}
      </Flex.Box>
      <Flex.Box x>
        <Form.SwitchField path={PATH.fd} label="CAN FD" />
        <Form.SwitchField path={PATH.listenOnly} label="Listen only" />
      </Flex.Box>
    </>
  );
};

export const { MAKE, useConnectModal, Select, COMMANDS, NETWORK } = Bus.createDevice({
  make: "CAN",
  name: "CAN",
  noun: "CAN device",
  description: "Read and write ICD messages on a CAN bus.",
  icon: <Icon.Hardware />,
  properties: can.propertiesZ,
  integration: "can",
  scanType: "can_scan",
  location: { shape: "plain" },
  Properties,
  getModel: ({ backend }) => backend,
  required: { channel: "Channel is required" },
  getLocation: ({ channel }) => channel,
});
