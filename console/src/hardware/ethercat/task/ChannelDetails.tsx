// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Flex, Form as PForm, Telem } from "@synnaxlabs/pluto";
import { type FC } from "react";

import { type Common } from "@/hardware/common";
import { SelectSlave } from "@/hardware/ethercat/device/SelectSlave";
import { SelectChannelModeField } from "@/hardware/ethercat/task/SelectChannelModeField";
import { SelectPDOField } from "@/hardware/ethercat/task/SelectPDOField";
import {
  AUTOMATIC_TYPE,
  type ChannelMode,
  type InputChannel,
  type OutputChannel,
  ZERO_READ_CHANNELS,
  ZERO_WRITE_CHANNELS,
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
        {({ value, onChange }) => (
          <Telem.SelectDataType value={value} onChange={onChange} hideVariableDensity />
        )}
      </PForm.Field>
    </Flex.Box>
  </>
);

export interface ChannelDetailsProps extends Common.Task.Layouts.DetailsProps {
  pdoType: "inputs" | "outputs";
  zeroChannels: Record<string, InputChannel | OutputChannel>;
}

export const ChannelDetails: FC<ChannelDetailsProps> = ({
  path,
  pdoType,
  zeroChannels,
}) => {
  const channelMode = PForm.useFieldValue<ChannelMode>(`${path}.type`);
  return (
    <Flex.Box y gap="medium" style={{ padding: "1rem" }}>
      <SelectSlave path={`${path}.device`} />
      <SelectChannelModeField path={path} zeroChannels={zeroChannels} />
      {channelMode === AUTOMATIC_TYPE ? (
        <SelectPDOField path={path} pdoType={pdoType} />
      ) : (
        <ManualChannelFields path={path} />
      )}
    </Flex.Box>
  );
};

export const ReadChannelDetails: FC<Common.Task.Layouts.DetailsProps> = (props) => (
  <ChannelDetails {...props} pdoType="inputs" zeroChannels={ZERO_READ_CHANNELS} />
);

export const WriteChannelDetails: FC<Common.Task.Layouts.DetailsProps> = (props) => (
  <ChannelDetails {...props} pdoType="outputs" zeroChannels={ZERO_WRITE_CHANNELS} />
);
