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
import { DataType, type deep } from "@synnaxlabs/x";
import { type ReactElement } from "react";

import { isMissing } from "@/platform/task/ChannelState";
import { dataTypeOf } from "@/platform/task/payload/kind";
import { type ReadOptions, readOptionsOf } from "@/platform/task/payload/read";

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

interface ValueProps {
  /** The data type of the channel that deploying creates. */
  value: string;
  /** Sets the data type, and with it the kind of the field. */
  onChange: (dataType: string) => void;
}

interface PathProps<F extends ReadOptions> {
  /** The path of a read field. */
  path: deep.Target<F>;
  /** Returns the field for the options of a new data type. */
  create: (field: F, options: ReadOptions) => F;
}

export type ReadDataTypeFieldProps<F extends ReadOptions = ReadOptions> = Pick<
  DataTypeFieldProps,
  "channel" | "hideDataTypes" | "hideVariableDensity"
> &
  (ValueProps | PathProps<F>);

/**
 * Selects the data type of the channel that a read field writes to, from every data
 * type unless hideDataTypes says otherwise. With a path, it also gives the field the
 * kind of that data type. An existing channel sets the data type, so the field then
 * shows that of the channel, locked.
 */
export const ReadDataTypeField = <F extends ReadOptions>(
  props: ReadDataTypeFieldProps<F>,
): ReactElement =>
  "path" in props ? <ReadPathDataType {...props} /> : <ReadValueDataType {...props} />;

const ReadPathDataType = <F extends ReadOptions>({
  path,
  create,
  ...rest
}: Omit<ReadDataTypeFieldProps<F>, keyof ValueProps> & PathProps<F>): ReactElement => {
  const { get, set } = Form.useContext();
  const field = Form.useFieldValue(path);
  const handleChange = (dataType: string) => {
    const current = get(path).value;
    set(path, create(current, readOptionsOf(dataType, current)));
  };
  return (
    <ReadValueDataType {...rest} value={dataTypeOf(field)} onChange={handleChange} />
  );
};

const ReadValueDataType = ({
  channel,
  hideDataTypes = NO_HIDDEN_DATA_TYPES,
  ...rest
}: Pick<DataTypeFieldProps, "channel" | "hideDataTypes" | "hideVariableDensity"> &
  ValueProps): ReactElement => {
  const unbound = (
    <Input.Item label="Data type" padHelpText={false}>
      <Select {...rest} hideDataTypes={hideDataTypes} />
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
  /** Renders in place of the locked field when the channel no longer exists. */
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
