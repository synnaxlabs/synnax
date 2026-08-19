// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type channel } from "@synnaxlabs/client";
import { type notation, primitive, type text } from "@synnaxlabs/x";
import { type ReactElement } from "react";

import { Channel } from "@/channel";
import { Component } from "@/component";
import { Flex } from "@/flex";
import { Form as Base } from "@/form";
import { Input } from "@/input";
import { Notation } from "@/notation";
import { Form as NodeForm } from "@/schematic/node/common/form";
import {
  type Config,
  createTelem,
  defaultConfig,
  parseTelem,
  type Side,
  type TelemProps,
} from "@/schematic/node/common/scale/config";
import { Select } from "@/select";
import { type telem } from "@/telem/aether";

const BOUND_INPUT_PROPS: Partial<Input.NumericProps> = { step: 10 };
const PRECISION_BOUNDS = { lower: 0, upper: 10 };
const WINDOW_SIZE_BOUNDS = { lower: 1, upper: 100 };

const SIDE_KEYS = ["left", "right"] as const;
const SideSelect = Component.renderProp(
  ({ value, onChange }: Input.Control<Side>): ReactElement => (
    <Select.Buttons value={value} onChange={onChange} keys={SIDE_KEYS}>
      <Select.Button itemKey="left">Left</Select.Button>
      <Select.Button itemKey="right">Right</Select.Button>
    </Select.Buttons>
  ),
);

export interface FormProps {
  /** Path to the scale config within the symbol's config. */
  path: string;
}

const field = (path: string, name: string): string => `${path}.${name}`;

export interface TelemFormProps extends FormProps {
  /** When true, clearing the channel unbinds the scale instead of pinning it to 0. */
  allowNone?: boolean;
  /** Applied when the symbol carries no scale config yet. */
  defaults?: Partial<Config>;
}

export const TelemForm = ({
  path,
  allowNone = false,
  defaults,
}: TelemFormProps): ReactElement => {
  const { set } = Base.useContext();
  const config = Base.useField<Config | undefined>(path, { optional: true })?.value;
  const props = parseTelem(config?.telem);
  const setTelem = (telem?: telem.StringSourceSpec): void => {
    if (config != null) return set(field(path, "telem"), telem);
    if (telem != null) set(path, defaultConfig({ ...defaults, telem }));
  };
  const handleChange = (next: Partial<TelemProps>): void =>
    setTelem(createTelem({ ...props, ...next }));
  const handleChannelChange = (key: channel.Key | null): void => {
    if (allowNone && !primitive.isNonZero(key)) return setTelem(undefined);
    handleChange({ channel: key ?? 0 });
  };
  return (
    <>
      <Flex.Box x>
        <Input.Item label="Channel" grow padHelpText={false}>
          <Channel.SelectSingle
            value={props.channel}
            onChange={handleChannelChange}
            allowNone={allowNone}
          />
        </Input.Item>
        {config != null && (
          <>
            <Base.NumericField
              path={field(path, "bounds.lower")}
              label="Min value"
              inputProps={BOUND_INPUT_PROPS}
              padHelpText={false}
            />
            <Base.NumericField
              path={field(path, "bounds.upper")}
              label="Max value"
              inputProps={BOUND_INPUT_PROPS}
              padHelpText={false}
            />
          </>
        )}
      </Flex.Box>
      {config != null && (
        <Flex.Box x>
          <Input.Item label="Notation">
            <Notation.Select
              value={props.notation}
              onChange={(notation: notation.Notation) => handleChange({ notation })}
            />
          </Input.Item>
          <Input.Item label="Precision" align="start">
            <Input.Numeric
              value={props.precision}
              bounds={PRECISION_BOUNDS}
              onChange={(precision) => handleChange({ precision })}
            />
          </Input.Item>
          <NodeForm.UnitsField path={field(path, "units")} />
          <Input.Item label="Averaging window" align="start" grow>
            <Input.Numeric
              value={props.windowSize}
              bounds={WINDOW_SIZE_BOUNDS}
              onChange={(windowSize) => handleChange({ windowSize })}
            />
          </Input.Item>
        </Flex.Box>
      )}
    </>
  );
};

export const Form = ({ path }: FormProps): ReactElement => (
  <Flex.Box x>
    <NodeForm.ColorField path={field(path, "color")} />
    <NodeForm.ColorField path={field(path, "axisColor")} label="Scale color" />
    <NodeForm.ColorField path={field(path, "textColor")} label="Text color" />
    <Base.SwitchField path={field(path, "showFill")} label="Fill" />
    <Base.SwitchField path={field(path, "showCaret")} label="Value" />
    <Base.SwitchField path={field(path, "showScale")} label="Scale" />
    <Base.Field<Side> path={field(path, "side")} label="Side" padHelpText={false}>
      {SideSelect}
    </Base.Field>
    <Base.Field<text.Level>
      path={field(path, "level")}
      label="Size"
      padHelpText={false}
    >
      {NodeForm.SelectTextLevel}
    </Base.Field>
  </Flex.Box>
);
