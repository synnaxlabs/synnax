// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { ethercat } from "@synnaxlabs/client";
import { Component } from "@synnaxlabs/lyra/component";
import { Form as PForm } from "@synnaxlabs/lyra/form";
import { Telem } from "@synnaxlabs/pluto";
import { primitive } from "@synnaxlabs/x";
import { type FC } from "react";
import { type z } from "zod";

import { Device } from "@/feature/ethercat/device";
import { SelectEntryField } from "@/feature/ethercat/task/SelectEntryField";
import {
  type Channel,
  entryAt,
  type EntryChannel,
  getEntryLabel,
  getPortLabel,
  hasEntry,
  type PDOType,
  READ_CHANNEL_SCHEMAS,
  WRITE_CHANNEL_SCHEMAS,
} from "@/feature/ethercat/task/types";
import { Task } from "@/platform/task";

const renderSelectDataType = Component.renderProp(
  (props: Telem.SelectDataTypeProps) => (
    <Telem.SelectDataType {...props} hideVariableDensity />
  ),
);

export interface ChannelDetailsProps extends Task.Views.DetailsProps {
  pdoType: PDOType;
  schema: z.ZodType<EntryChannel>;
}

const ChannelDetails: FC<ChannelDetailsProps> = ({ path, pdoType, schema }) => {
  const ch = PForm.useFieldValue<Channel>(path);
  return (
    <PForm.Sections>
      <PForm.Section title="Source">
        <Device.SelectSlave path={`${path}.device`} />
      </PForm.Section>
      <PForm.Section title="Entry">
        <SelectEntryField path={path} pdoType={pdoType} schema={schema} />
        {hasEntry(ch) && (
          <PForm.Field<string> path={`${path}.dataType`} label="Data type">
            {renderSelectDataType}
          </PForm.Field>
        )}
      </PForm.Section>
    </PForm.Sections>
  );
};

export const ReadChannelDetails: FC<Task.Views.DetailsProps> = (props) => (
  <ChannelDetails {...props} pdoType="inputs" schema={READ_CHANNEL_SCHEMAS.entry} />
);

export const WriteChannelDetails: FC<Task.Views.DetailsProps> = (props) => (
  <ChannelDetails {...props} pdoType="outputs" schema={WRITE_CHANNEL_SCHEMAS.entry} />
);

interface DetailsTitleProps extends Task.Views.DetailsProps {
  pdoType: PDOType;
}

const DetailsTitle: FC<DetailsTitleProps> = ({ path, pdoType }) => {
  const ch = PForm.useFieldValue<Channel>(path);
  const { data: slave } = Device.useResultSlave(
    primitive.isZero(ch.device) ? null : { key: ch.device },
  );
  let title = getPortLabel(ch);
  if (hasEntry(ch)) {
    const entry = entryAt(slave?.properties.pdos[pdoType] ?? [], ch);
    if (entry != null)
      title = `${getEntryLabel(entry)} · ${ethercat.formatAddress(ch)}`;
  }
  return <Task.Views.ItemLabel kind="PDO">{title}</Task.Views.ItemLabel>;
};

export const ReadDetailsTitle: FC<Task.Views.DetailsProps> = (props) => (
  <DetailsTitle {...props} pdoType="inputs" />
);

export const WriteDetailsTitle: FC<Task.Views.DetailsProps> = (props) => (
  <DetailsTitle {...props} pdoType="outputs" />
);
