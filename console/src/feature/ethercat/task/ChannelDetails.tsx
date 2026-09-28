// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Component } from "@synnaxlabs/lyra/component";
import { Form as PForm } from "@synnaxlabs/lyra/form";
import { Telem } from "@synnaxlabs/pluto";
import { type FC } from "react";

import { SelectSlave } from "@/feature/ethercat/device/SelectSlave";
import { SelectChannelModeField } from "@/feature/ethercat/task/SelectChannelModeField";
import { SelectPDOField } from "@/feature/ethercat/task/SelectPDOField";
import {
  type ChannelMode,
  type ChannelSchemas,
  READ_CHANNEL_SCHEMAS,
  WRITE_CHANNEL_SCHEMAS,
} from "@/feature/ethercat/task/types";
import { type Task } from "@/platform/task";

const INPUT_PROPS = { showDragHandle: false };

const ManualChannelFields: FC<{ path: string }> = ({ path }) => (
  <>
    <PForm.NumericField
      path={`${path}.index`}
      label="Index (hex)"
      inputProps={INPUT_PROPS}
    />
    <PForm.NumericField
      path={`${path}.subIndex`}
      label="Subindex"
      inputProps={INPUT_PROPS}
    />
    <PForm.NumericField
      path={`${path}.bitLength`}
      label="Bit length"
      inputProps={INPUT_PROPS}
    />
    <PForm.Field<string> path={`${path}.dataType`} label="Data type">
      {renderSelectDataType}
    </PForm.Field>
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
    <PForm.Sections>
      <PForm.Section title="Source">
        <SelectSlave path={`${path}.device`} />
        <SelectChannelModeField path={path} schemas={schemas} />
      </PForm.Section>
      <PForm.Section title="Entry">
        {channelMode === "automatic" ? (
          <SelectPDOField path={path} pdoType={pdoType} />
        ) : (
          <ManualChannelFields path={path} />
        )}
      </PForm.Section>
    </PForm.Sections>
  );
};

export const ReadChannelDetails: FC<Task.Views.DetailsProps> = (props) => (
  <ChannelDetails {...props} pdoType="inputs" schemas={READ_CHANNEL_SCHEMAS} />
);

export const WriteChannelDetails: FC<Task.Views.DetailsProps> = (props) => (
  <ChannelDetails {...props} pdoType="outputs" schemas={WRITE_CHANNEL_SCHEMAS} />
);
