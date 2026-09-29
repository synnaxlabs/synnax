// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { mil1553 } from "@synnaxlabs/client";
import { Flex } from "@synnaxlabs/lyra/flex";
import { Form } from "@synnaxlabs/lyra/form";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Select as LyraSelect } from "@synnaxlabs/lyra/select";
import { type ReactElement } from "react";

import { Bus } from "@/feature/bus";
import { Task } from "@/platform/task";

const BACKEND_NAMES = {
  simulated: "Simulated",
  ddc: "DDC",
  ballard: "Astronics Ballard",
} as const satisfies Record<mil1553.Backend, string>;

// The Driver cannot drive a DDC or Ballard card yet.
const OFFERED_BACKENDS: mil1553.Backend[] = ["simulated"];

const ROLE_NAMES = {
  bus_controller: "Bus controller",
  remote_terminal: "Remote terminal",
  monitor: "Monitor",
} as const satisfies Record<mil1553.Role, string>;

const TERMINALS_PATH = "properties.terminals";

const RoleField = Form.buildSelectField<mil1553.Role>({
  fieldKey: "role",
  fieldProps: {
    label: "Role",
    // Only a remote terminal owns terminals.
    onChange: (role, { set }) => {
      if (role !== "remote_terminal") set(TERMINALS_PATH, []);
    },
  },
  inputProps: { resourceName: "role", children: Task.selectItems(ROLE_NAMES) },
});

// Remote terminal addresses. 31 is the broadcast address.
const TERMINAL_ITEMS = Array.from({ length: 31 }, (_, rt) => (
  <LyraSelect.Item<number> key={rt} itemKey={rt}>
    {rt.toString()}
  </LyraSelect.Item>
));

const byAddress = (a: number, b: number) => a - b;

const Properties = (): ReactElement => {
  const role = Form.useFieldValue<mil1553.Role>("properties.role");
  return (
    <>
      <Bus.BackendField names={BACKEND_NAMES} offered={OFFERED_BACKENDS} />
      <Flex.Box x>
        <Form.NumericField path="properties.card" label="Card" />
        <Form.NumericField path="properties.channel" label="Channel" />
        <RoleField path="properties" />
      </Flex.Box>
      <Form.Field<number[]>
        path={TERMINALS_PATH}
        label="Terminals"
        visible={role === "remote_terminal"}
      >
        {({ value, onChange, preview }) => (
          <LyraSelect.Simple<number>
            multiple
            resourceName="terminal"
            value={value}
            onChange={(next) => onChange([...next].sort(byAddress))}
            preview={preview}
          >
            {TERMINAL_ITEMS}
          </LyraSelect.Simple>
        )}
      </Form.Field>
    </>
  );
};

export const { MAKE, useConnectModal, Select, COMMANDS } = Bus.createDevice({
  make: "MIL-STD-1553",
  noun: "MIL-STD-1553 device",
  icon: <Icon.Node />,
  properties: mil1553.propertiesZ,
  integration: "mil1553",
  Properties,
  getModel: ({ backend }) => backend,
  getLocation: ({ card, channel }) => `Card ${card}, channel ${channel}`,
  validate: ({ role, terminals }) =>
    role === "remote_terminal" && terminals.length === 0
      ? { terminals: "Select at least one terminal" }
      : {},
});
