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
  hideDataTypes = HIDDEN_DATA_TYPES,
  hideVariableDensity,
}: DataTypeFieldProps): ReactElement => {
  const unbound = (
    <Form.Field path={path} label="Data type" padHelpText={false}>
      {(p) => (
        <Telem.SelectDataType
          {...p}
          hideDataTypes={hideDataTypes}
          hideVariableDensity={hideVariableDensity}
          location="bottom"
        />
      )}
    </Form.Field>
  );
  return channel === 0 ? unbound : <Bound channel={channel} unbound={unbound} />;
};

export interface ReadDataTypeFieldProps {
  /** The data type of the channel that deploying creates. */
  value: string;
  /** Sets the data type, and with it the kind of the field. */
  onChange: (dataType: string) => void;
  /** The channel that sets the data type, or 0 before deploying creates one. */
  channel: channel.Key;
}

/**
 * Selects the data type of the channel that a read field writes to, from every data
 * type. An existing channel sets the data type, so the field then shows that of the
 * channel, locked.
 */
export const ReadDataTypeField = ({
  value,
  onChange,
  channel,
}: ReadDataTypeFieldProps): ReactElement => {
  const unbound = (
    <Input.Item label="Data type" padHelpText={false}>
      <Telem.SelectDataType
        value={value}
        onChange={onChange}
        hideDataTypes={NO_HIDDEN_DATA_TYPES}
        location="bottom"
      />
    </Input.Item>
  );
  return channel === 0 ? unbound : <Bound channel={channel} unbound={unbound} />;
};

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
