// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ReactElement } from "react";

import { Text } from "@synnaxlabs/pluto/text";

import { type PhysicalGroupPlan } from "./types";

export interface GroupDocProps {
  group: PhysicalGroupPlan;
}

export type GroupDoc = (props: GroupDocProps) => ReactElement;

const DigitalIOCommandGroupDoc: GroupDoc = ({ group }) => {
  const { name } = group;
  const portAndLine = name.split(" ").slice(-1)[0];
  const [port, line] = portAndLine.split("/").slice(-2);

  return (
    <Text.Text level="p">
      This group is used to send commands to the dual purpose digital input/output
      physical channel port {port} line {line}. By default, it contains two channels.
      The first channel is used to send the actual command value, and the second channel
      is used to store the timestamp at which the command was issued. When a task
      configures this physical channel as an output, its corresponding digital input
      channel will be used to acknowledge the commanded state. When the physical channel
      is configured to be an input, this group will have no effect on operations.
    </Text.Text>
  );
};

const DigitalOutGroupDoc: GroupDoc = ({ group }) => {
  const { name } = group;
  const portAndLine = name.split(" ").slice(-1)[0];
  const [port, line] = portAndLine.split("/").slice(-2);

  return (
    <Text.Text level="p">
      This group is used to send commands to the digital output physical channel on port{" "}
      {port} line {line}. By default, it contains two channels. The first channel is
      used to send the actual command value, and the second channel is used to store the
      timestamp at which the command was issued. Its corresponding acknowledgement
      channel in group {"HERE"} will be used to acknowledge the commanded state.
    </Text.Text>
  );
};

const DigitalIOInputGroupDoc: GroupDoc = ({ group }) => {
  return (
    <Text.Text level="p">
      This group is used to read the dual purpose digital/input channels on your device.
      All of the channels in this group are sampled together, and the timestamps of
      their readings are stored in this groups index channel. This means that the
      channels in this task cannot be sampled at different rates. If you need to sample
      at different rates, split the channels into separate groups. Although they must be
      sampled at the same rate, acquisition from chosen channels can be turned off when
      not in use. When the physical channels are configured to be an output, the digital
      input channel will be used to acknolwedge the commanded state. Please note that
      all the channels in this group must be configured to be inputs or outputs, not a
      mix. If you need to mix, split the channels into separate groups.
    </Text.Text>
  );
};

const DigitalInputGroup: GroupDoc = ({ group }) => {
  return (
    <Text.Text level="p">
      This group is used to read data digital inputs on your device. All of the channels
      in this group are sampled together, and the timestamps of their readings are
      stored in this groups index channel. This means that the channels in this task
      cannot be sampled at different rates. If you need to sample at different rates,
      split the channels into separate groups. Although they must be sampled at the same
      rate, acquisition from chosen channels can be turned off when not in use.
    </Text.Text>
  );
};

const DigitalOutputAckGroupDoc: GroupDoc = ({ group }) => {
  return (
    <Text.Text level="p">
      This group is used to acknowledge the commanded state of the digital output
      channels on this device. The channels in this group are sampled together, and the
      timestamps of their readings are stored in this groups index channel. This means
      that sending a command to one of the digital output channels will also sample the
      states of all the digital output channels.
    </Text.Text>
  );
};
