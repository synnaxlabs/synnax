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
import { DataType } from "@synnaxlabs/x";
import { type ReactElement } from "react";

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
  path: string;
  /** The channel that sets the data type, or 0 before configure creates one. */
  channel: channel.Key;
}

/**
 * Selects the data type of the channel that configure creates. A bound channel sets the
 * data type, so the field then shows that of the channel, locked.
 */
export const DataTypeField = ({
  path,
  channel,
  hideDataTypes = HIDDEN_DATA_TYPES,
  hideVariableDensity,
}: DataTypeFieldProps): ReactElement => {
  if (channel !== 0) return <Bound channel={channel} />;
  return (
    <Form.Field<string> path={path} label="Data type" padHelpText={false}>
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
};

const NO_HIDDEN_DATA_TYPES: DataType[] = [];

const Bound = ({ channel }: Pick<DataTypeFieldProps, "channel">): ReactElement => {
  const { data } = Channel.useResultDataType({ key: channel });
  return (
    <Input.Item label="Data type" padHelpText={false} helpText="Set on the channel">
      <Telem.SelectDataType
        value={data?.toString() ?? ""}
        onChange={NO_CHANGE}
        hideDataTypes={NO_HIDDEN_DATA_TYPES}
        disabled
        location="bottom"
      />
    </Input.Item>
  );
};

const NO_CHANGE = () => {};
