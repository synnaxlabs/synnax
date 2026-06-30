// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Component, Flex, Form as PForm, Telem } from "@synnaxlabs/pluto";
import { type FC } from "react";

import { type Common } from "@/hardware/common";
import { SelectSlave } from "@/hardware/ethercat/device/SelectSlave";
import { SelectChannelModeField } from "@/hardware/ethercat/task/SelectChannelModeField";
import { SelectPDOField } from "@/hardware/ethercat/task/SelectPDOField";
import {
  type Channel,
  type ChannelMode,
  ZERO_INPUT_CHANNELS,
  ZERO_OUTPUT_CHANNELS,
} from "@/hardware/ethercat/task/types";

const INPUT_PROPS = { showDragHandle: false };

const ManualChannelFields: FC<{ path: string }> = ({ path }) => (
  <>
    <Flex.Box x gap="small">
      <PForm.NumericField
        path={`${path}.index`}
        label="Index (hex)"
        inputProps={INPUT_PROPS}
        grow
      />
      <PForm.NumericField
        path={`${path}.subIndex`}
        label="Subindex"
        inputProps={INPUT_PROPS}
        grow
      />
    </Flex.Box>
    <Flex.Box x gap="small">
      <PForm.NumericField
        path={`${path}.bitLength`}
        label="Bit Length"
        inputProps={INPUT_PROPS}
        grow
      />
      <PForm.Field<string> path={`${path}.dataType`} label="Data Type" grow>
        {renderSelectDataType}
      </PForm.Field>
    </Flex.Box>
  </>
);

const renderSelectDataType = Component.renderProp(
  (props: Telem.SelectDataTypeProps) => (
    <Telem.SelectDataType {...props} hideVariableDensity />
  ),
);

export interface ChannelDetailsProps extends Common.Task.Layouts.DetailsProps {
  pdoType: "inputs" | "outputs";
  zeroChannels: Record<ChannelMode, Channel>;
}

const ChannelDetails: FC<ChannelDetailsProps> = ({ path, pdoType, zeroChannels }) => {
  const channelMode = PForm.useFieldValue<ChannelMode>(`${path}.type`);
  return (
    <Flex.Box y gap="medium" style={CHANNEL_DETAILS_STYLE}>
      <SelectSlave path={`${path}.device`} />
      <SelectChannelModeField path={path} zeroChannels={zeroChannels} />
      {channelMode === "automatic" ? (
        <SelectPDOField path={path} pdoType={pdoType} />
      ) : (
        <ManualChannelFields path={path} />
      )}
    </Flex.Box>
  );
};

const CHANNEL_DETAILS_STYLE = { padding: "1rem" } as const;

export const ReadChannelDetails: FC<Common.Task.Layouts.DetailsProps> = (props) => (
  <ChannelDetails {...props} pdoType="inputs" zeroChannels={ZERO_INPUT_CHANNELS} />
);

export const WriteChannelDetails: FC<Common.Task.Layouts.DetailsProps> = (props) => (
  <ChannelDetails {...props} pdoType="outputs" zeroChannels={ZERO_OUTPUT_CHANNELS} />
);
