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

import { SelectSlave } from "@/feature/ethercat/device/SelectSlave";
import { SelectChannelModeField } from "@/feature/ethercat/task/SelectChannelModeField";
import { SelectPDOField } from "@/feature/ethercat/task/SelectPDOField";
import {
  type ChannelMode,
  type ChannelSchemas,
  INPUT_CHANNEL_SCHEMAS,
  OUTPUT_CHANNEL_SCHEMAS,
} from "@/feature/ethercat/task/types";
import { type Task } from "@/platform/task";

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

export interface ChannelDetailsProps extends Task.Views.DetailsProps {
  pdoType: "inputs" | "outputs";
  schemas: ChannelSchemas;
}

const ChannelDetails: FC<ChannelDetailsProps> = ({ path, pdoType, schemas }) => {
  const channelMode = PForm.useFieldValue<ChannelMode>(`${path}.type`);
  return (
    <Flex.Box y gap="medium" style={CHANNEL_DETAILS_STYLE}>
      <SelectSlave path={`${path}.device`} />
      <SelectChannelModeField path={path} schemas={schemas} />
      {channelMode === "automatic" ? (
        <SelectPDOField path={path} pdoType={pdoType} />
      ) : (
        <ManualChannelFields path={path} />
      )}
    </Flex.Box>
  );
};

const CHANNEL_DETAILS_STYLE = { padding: "1rem" } as const;

export const ReadChannelDetails: FC<Task.Views.DetailsProps> = (props) => (
  <ChannelDetails {...props} pdoType="inputs" schemas={INPUT_CHANNEL_SCHEMAS} />
);

export const WriteChannelDetails: FC<Task.Views.DetailsProps> = (props) => (
  <ChannelDetails {...props} pdoType="outputs" schemas={OUTPUT_CHANNEL_SCHEMAS} />
);
