// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type channel } from "@synnaxlabs/client";
import { Form } from "@synnaxlabs/lyra/form";
import { Input } from "@synnaxlabs/lyra/input";
import { Channel, Telem } from "@synnaxlabs/pluto";
import { DataType, deep } from "@synnaxlabs/x";
import { type ReactElement } from "react";

import { isMissing } from "@/platform/task/ChannelState";
import {
  dataTypeOf,
  type ReadOptions,
  readOptionsOf,
} from "@/platform/task/payload/read";

/** Data types that no JSON field can carry. */
const HIDDEN_DATA_TYPES = [
  DataType.TIMESTAMP,
  DataType.UUID,
  DataType.JSON,
  DataType.BYTES,
];

export interface DataTypeFieldProps extends Pick<
  Telem.SelectDataTypeProps,
  "hideDataTypes" | "hideVariableDensity"
> {
  path: deep.Target<string>;
  /** The channel that sets the data type, or 0 before deploying creates one. */
  channel: channel.Key;
}

/**
 * Selects the data type of the channel that deploying creates. An existing channel sets
 * the data type, so the field then shows that of the channel, locked.
 */
export const DataTypeField = ({
  channel,
  path,
  ...rest
}: DataTypeFieldProps): ReactElement => {
  const unbound = (
    <Form.Field path={path} label="Data type" padHelpText={false}>
      {(p) => <Select {...p} {...rest} />}
    </Form.Field>
  );
  return channel === 0 ? unbound : <Bound channel={channel} unbound={unbound} />;
};

export interface ReadDataTypeFieldProps<F extends ReadOptions> extends Pick<
  DataTypeFieldProps,
  "channel" | "hideDataTypes" | "hideVariableDensity"
> {
  /** The path of a read field. */
  path: deep.Target<F>;
  /** Returns the field for the options of a new data type. */
  create: (field: F, options: ReadOptions) => F;
}

/**
 * Selects the data type of the channel that deploying creates for a read field, and
 * gives the field the kind of that data type. An existing channel sets the data type,
 * so the field then shows that of the channel, locked.
 */
export const ReadDataTypeField = <F extends ReadOptions>({
  path,
  channel,
  create,
  ...rest
}: ReadDataTypeFieldProps<F>): ReactElement => {
  const { get, set } = Form.useContext();
  const field = Form.useFieldValue(path);
  // The deploy check reports a kind that does not suit the field at its type.
  const kind = Form.useFieldState(deep.widen<ReadOptions>(path).type, {
    optional: true,
  });
  const handleChange = (dataType: string) => {
    const current = get(path).value;
    set(path, create(current, readOptionsOf(dataType, current)));
  };
  const unbound = (
    <Input.Item
      label="Data type"
      padHelpText={false}
      helpText={kind?.status.message}
      status={kind?.status.variant}
    >
      <Select value={dataTypeOf(field)} onChange={handleChange} {...rest} />
    </Input.Item>
  );
  return channel === 0 ? unbound : <Bound channel={channel} unbound={unbound} />;
};

const Select = ({
  hideDataTypes = HIDDEN_DATA_TYPES,
  ...rest
}: Telem.SelectDataTypeProps): ReactElement => (
  <Telem.SelectDataType {...rest} hideDataTypes={hideDataTypes} location="bottom" />
);

const NO_HIDDEN_DATA_TYPES: DataType[] = [];

interface BoundProps {
  channel: channel.Key;
  /** Renders once the channel no longer exists. */
  unbound: ReactElement;
}

const Bound = ({ channel, unbound }: BoundProps): ReactElement => {
  const result = Channel.useResultDataType({ key: channel });
  if (isMissing(result)) return unbound;
  return (
    <Input.Item label="Data type" padHelpText={false} helpText="Set on the channel">
      <Telem.SelectDataType
        value={result.data?.toString() ?? ""}
        onChange={NO_CHANGE}
        hideDataTypes={NO_HIDDEN_DATA_TYPES}
        disabled
        location="bottom"
      />
    </Input.Item>
  );
};

const NO_CHANGE = () => {};
