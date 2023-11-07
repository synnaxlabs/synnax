// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ReactElement } from "react";

import { type channel } from "@synnaxlabs/client";

import { Channel } from "@/channel";
import { Input } from "@/input";
import { type remote } from "@/telem/remote/aether";
import { componentRenderProp } from "@/util/renderProp";

export interface NumericSourceFormProps
  extends Omit<
    Input.ItemProps<Omit<remote.NumericSourceProps, "units" | "precision">>,
    "children"
  > {}

export const NumericSourceForm = ({
  value,
  onChange,
  ...props
}: NumericSourceFormProps): ReactElement => {
  const handleChannelChange = (channel: channel.Key): void =>
    onChange({ ...value, channel });

  return (
    <Input.Item<channel.Key, channel.Key, Channel.SelectSingleProps>
      label="Channel"
      value={value.channel}
      onChange={handleChannelChange}
      {...props}
    >
      {componentRenderProp(Channel.SelectSingle)}
    </Input.Item>
  );
};

export interface NumericStringSourceFormProps
  extends Input.ItemProps<remote.NumericSourceProps> {}

export const NumericStringSourceForm = ({
  value,
  onChange,
  ...props
}: NumericStringSourceFormProps): ReactElement => {
  const handleBaseChange = (props: Pick<remote.NumericSourceProps, "channel">): void =>
    onChange({ ...value, ...props });

  return (
    <>
      <NumericSourceForm {...props} value={value} onChange={handleBaseChange} />
      <Input.Item<string>
        label="Units"
        value={value.units}
        onChange={(units): void => onChange({ ...value, units })}
      />
      <Input.Item<number, number, Input.NumericProps>
        label="Precision"
        value={value.precision}
        onChange={(precision: number): void => onChange({ ...value, precision })}
        bounds={{ lower: 0, upper: 10 }}
      >
        {componentRenderProp(Input.Numeric)}
      </Input.Item>
    </>
  );
};
