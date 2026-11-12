// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type device, ethercat } from "@synnaxlabs/client";
import { Component } from "@synnaxlabs/lyra/component";
import { Form as PForm } from "@synnaxlabs/lyra/form";
import { Status } from "@synnaxlabs/lyra/status";
import { Text } from "@synnaxlabs/lyra/text";
import { Synnax, Telem } from "@synnaxlabs/pluto";
import { deep, primitive } from "@synnaxlabs/x";
import { type FC, type ReactNode } from "react";
import { type z } from "zod";

import { useResultSlave } from "@/feature/ethercat/device/queries";
import {
  SelectSlave,
  type SelectSlaveProps,
} from "@/feature/ethercat/device/SelectSlave";
import { of } from "@/feature/ethercat/device/setup";
import { SetupNote } from "@/feature/ethercat/device/SetupNote";
import { SLAVE_SCHEMAS } from "@/feature/ethercat/device/types";
import { ChannelSettings } from "@/feature/ethercat/task/ChannelSettings";
import { SelectEntryField } from "@/feature/ethercat/task/SelectEntryField";
import {
  type Channel,
  entryAt,
  type EntryChannel,
  getEntryLabel,
  getPortLabel,
  hasEntry,
  isCalibrated,
  type PDOType,
  READ_CHANNEL_SCHEMAS,
  type ReadChannel,
  type ReadSchemas,
  WRITE_CHANNEL_SCHEMAS,
  type WriteChannel,
} from "@/feature/ethercat/task/types";
import { Task } from "@/platform/task";

const renderSelectDataType = Component.renderProp(
  (props: Telem.SelectDataTypeProps) => (
    <Telem.SelectDataType {...props} hideVariableDensity />
  ),
);

export interface ChannelDetailsProps {
  path: deep.Path<Channel>;
  pdoType: PDOType;
  schema: z.ZodType<EntryChannel>;
  /** Fields shown after the data type while the channel has an entry. */
  entryFields?: ReactNode;
  /** Sections shown after the entry while the channel has one. */
  children?: ReactNode;
  /** Runs after the device of the channel changes. */
  onDeviceChange?: SelectSlaveProps["onChange"];
}

const ChannelDetails: FC<ChannelDetailsProps> = ({
  path,
  pdoType,
  schema,
  entryFields,
  children,
  onDeviceChange,
}) => {
  const ch = PForm.useFieldValue(path);
  const calibrated = hasEntry(ch) && "calibrated" in ch && ch.calibrated;
  const { data: slave } = useResultSlave(
    primitive.isZero(ch.device) ? null : { key: ch.device },
  );
  // Synnax reads the data of a slave that other software sets up, so it needs no note.
  const noted = slave != null && of(slave.properties).kind !== "shared";
  return (
    <PForm.Sections>
      <PForm.Section title="Entry">
        <SelectSlave path={path.device} onChange={onDeviceChange} />
        {noted && <SetupNote deviceKey={ch.device} aligned />}
        <SelectEntryField path={path} pdoType={pdoType} schema={schema} />
        {hasEntry(ch) && !calibrated && (
          <PForm.Field
            path={deep.variant(path, "type", "entry").dataType}
            label="Data type"
          >
            {renderSelectDataType}
          </PForm.Field>
        )}
        {hasEntry(ch) && entryFields}
      </PForm.Section>
      {hasEntry(ch) && children}
    </PForm.Sections>
  );
};

/**
 * @returns what runs after the device of the read channel changes: it sets whether the
 * Driver calibrates the entry of the channel on the new slave.
 */
const useRecalibrate = (
  path: deep.Path<ReadChannel>,
): ((key: device.Key, ctx: PForm.ContextValue) => void) => {
  const client = Synnax.use();
  const handleError = Status.useErrorHandler();
  return (key, { get, set }) => {
    if (client == null || primitive.isZero(key)) return;
    handleError(async () => {
      const slave = await client.devices.retrieve({ key, schemas: SLAVE_SCHEMAS });
      const ch = get<ReadChannel>(path).value;
      if (!hasEntry(ch) || ch.device !== key) return;
      set(
        deep.variant(path, "type", "entry").calibrated,
        isCalibrated(slave.properties, ch),
      );
    }, "Failed to retrieve device");
  };
};

export const ReadChannelDetails: FC<Task.Views.DetailsProps<ReadChannel>> = ({
  path,
}) => {
  const recalibrate = useRecalibrate(path);
  const { key } = PForm.useFieldValue(path);
  return (
    <ChannelDetails
      path={path}
      pdoType="inputs"
      schema={READ_CHANNEL_SCHEMAS.entry}
      onDeviceChange={recalibrate}
    >
      <ChannelSettings keys={[key]} />
      <PForm.Section title="Scale">
        <Task.ScaleForm path={deep.variant(path, "type", "entry").scale} />
      </PForm.Section>
    </ChannelDetails>
  );
};

const READ_CHANNELS = deep.path<Task.Values<ReadSchemas>>().config.channels;

/** Edits the settings and scale of the selected read channels that have an entry. */
export const ReadMultiDetails: FC<Task.Views.MultiDetailsProps> = ({ keys }) => {
  const channels = PForm.useFieldValue(READ_CHANNELS);
  const paths = channels
    .filter((c) => keys.includes(c.key) && hasEntry(c))
    .map((c) => deep.variant(READ_CHANNELS[c.key], "type", "entry"));
  if (paths.length === 0) return null;
  return (
    <PForm.Sections>
      <ChannelSettings keys={keys} />
      <PForm.Section title="Scale">
        <PForm.Fanout paths={paths}>
          <Task.ScaleForm path={paths[0].scale} />
        </PForm.Fanout>
      </PForm.Section>
    </PForm.Sections>
  );
};

export const WriteChannelDetails: FC<Task.Views.DetailsProps<WriteChannel>> = ({
  path,
}) => (
  <ChannelDetails
    path={path}
    pdoType="outputs"
    schema={WRITE_CHANNEL_SCHEMAS.entry}
    entryFields={
      <PForm.NumericField
        path={deep.variant(path, "type", "entry").stopValue}
        label="Stop value"
      />
    }
  />
);

interface DetailsTitleProps {
  path: deep.Target<Channel>;
  pdoType: PDOType;
}

const DetailsTitle: FC<DetailsTitleProps> = ({ path, pdoType }) => {
  const ch = PForm.useFieldValue(path);
  const { data: slave } = useResultSlave(
    primitive.isZero(ch.device) ? null : { key: ch.device },
  );
  let title = getPortLabel(ch);
  if (hasEntry(ch)) {
    const entry = entryAt(slave?.properties.pdos[pdoType] ?? [], ch);
    if (entry != null)
      title = `${getEntryLabel(entry)} · ${ethercat.formatAddress(ch)}`;
  }
  return (
    <Text.Text weight={500} color={10} overflow="fade">
      {title}
    </Text.Text>
  );
};

export const ReadDetailsTitle: FC<Task.Views.DetailsProps<ReadChannel>> = (props) => (
  <DetailsTitle {...props} pdoType="inputs" />
);

export const WriteDetailsTitle: FC<Task.Views.DetailsProps<WriteChannel>> = (props) => (
  <DetailsTitle {...props} pdoType="outputs" />
);
