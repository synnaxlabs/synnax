// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { arinc429 } from "@synnaxlabs/client";
import { Flex } from "@synnaxlabs/lyra/flex";
import { Form } from "@synnaxlabs/lyra/form";
import { Icon } from "@synnaxlabs/lyra/icon";
import { type ReactElement } from "react";

import { Bus } from "@/feature/bus";
import { Task } from "@/platform/task";

const BACKEND_NAMES = {
  simulated: "Simulated",
  ballard: "Astronics Ballard",
  ddc: "DDC",
} as const satisfies Record<arinc429.Backend, string>;

const SPEED_NAMES = {
  low: "Low (12.5 kbit/s)",
  high: "High (100 kbit/s)",
} as const satisfies Record<arinc429.Speed, string>;

const BackendField = Form.buildSelectField<arinc429.Backend>({
  fieldKey: "backend",
  fieldProps: { label: "Backend" },
  inputProps: { resourceName: "backend", children: Task.selectItems(BACKEND_NAMES) },
});

const SpeedField = Form.buildSelectField<arinc429.Speed>({
  fieldKey: "speed",
  fieldProps: { label: "Speed" },
  inputProps: { resourceName: "speed", children: Task.selectItems(SPEED_NAMES) },
});

const Properties = (): ReactElement => (
  <>
    <BackendField path="properties" />
    <Flex.Box x>
      <Form.NumericField path="properties.card" label="Card" />
      <Form.NumericField path="properties.channel" label="Channel" />
      <SpeedField path="properties" />
    </Flex.Box>
  </>
);

export const {
  MAKE,
  SCHEMAS,
  use,
  useResult,
  useFromConfig,
  useConnectModal,
  Select,
  COMMANDS,
} = Bus.createDevice({
  make: "ARINC 429",
  noun: "ARINC 429 device",
  icon: <Icon.Wave.Square />,
  properties: arinc429.propertiesZ,
  integration: "arinc429",
  Properties,
  getModel: ({ backend }) => backend,
  getLocation: ({ card, channel }) => `Card ${card}, channel ${channel}`,
});
