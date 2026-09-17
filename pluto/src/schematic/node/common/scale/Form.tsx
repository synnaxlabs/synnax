// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type channel } from "@synnaxlabs/client";
import {
  caseconv,
  type direction,
  location,
  type notation,
  primitive,
  type text,
} from "@synnaxlabs/x";
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
  DEFAULT_SIDE,
  defaultConfig,
  parseTelem,
  type TelemProps,
} from "@/schematic/node/common/scale/config";
import { Select } from "@/select";
import { type telem } from "@/telem/aether";
import { Staleness } from "@/vis/staleness";

const PRECISION_INPUT_PROPS: Partial<Input.NumericProps> = {
  bounds: { lower: 0, upper: 10 },
};
const WINDOW_SIZE_BOUNDS = { lower: 1, upper: 100 };

const NotationSelect = Component.renderProp(
  ({ value, onChange }: Input.Control<notation.Notation>): ReactElement => (
    <Notation.Select value={value} onChange={onChange} />
  ),
);

const SIDES: readonly location.Outer[] = [
  ...location.Y_LOCATIONS,
  ...location.X_LOCATIONS,
];

export interface FormProps {
  /** Path to the scale config within the symbol's config. */
  path: string;
}

const field = (path: string, name: string): string => `${path}.${name}`;

interface SideFieldProps {
  path: string;
  label: string;
  /** The sides to offer. A symbol with a fixed axis offers only the two it can use. */
  sides: readonly location.Outer[];
}

// A field on the other axis takes the side facing the same way as the default.
const defaultSide = (sides: readonly location.Outer[]): location.Outer =>
  sides.includes(DEFAULT_SIDE) ? DEFAULT_SIDE : location.swapAxis(DEFAULT_SIDE);

const SideField = ({ path, label, sides }: SideFieldProps): ReactElement => (
  <Base.Field<location.Outer>
    path={path}
    label={label}
    padHelpText={false}
    defaultValue={defaultSide(sides)}
  >
    {({ value, onChange }) => (
      <Select.Buttons value={value} onChange={onChange} keys={sides}>
        {sides.map((side) => (
          <Select.Button key={side} itemKey={side}>
            {caseconv.capitalize(side)}
          </Select.Button>
        ))}
      </Select.Buttons>
    )}
  </Base.Field>
);

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
  const setTelem = (telem?: telem.NumberSourceSpec): void => {
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
          <NodeForm.BoundsFields path={field(path, "bounds")} padHelpText={false} />
        )}
      </Flex.Box>
      {config != null && (
        <Flex.Box x>
          <Base.Field<notation.Notation>
            path={field(path, "notation")}
            label="Notation"
            padHelpText={false}
          >
            {NotationSelect}
          </Base.Field>
          <Base.NumericField
            path={field(path, "precision")}
            label="Precision"
            align="start"
            padHelpText={false}
            inputProps={PRECISION_INPUT_PROPS}
          />
          <NodeForm.UnitsField path={field(path, "units")} />
          <Staleness.Fields path={path} />
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

export interface DisplayFieldsProps extends FormProps {
  /** The axis the bar fills along, which the ticks must sit clear of. */
  axis?: direction.Direction;
}

/** Which parts of the scale are drawn, and the sides the ticks and readout sit on. */
export const DisplayFields = ({
  path,
  axis = "y",
}: DisplayFieldsProps): ReactElement => (
  <>
    <Base.SwitchField path={field(path, "showFill")} label="Fill" padHelpText={false} />
    <Base.SwitchField
      path={field(path, "showCaret")}
      label="Value"
      padHelpText={false}
    />
    <Base.SwitchField
      path={field(path, "showScale")}
      label="Scale"
      padHelpText={false}
    />
    <SideField path={field(path, "caretSide")} label="Value side" sides={SIDES} />
    <SideField
      path={field(path, "side")}
      label="Scale side"
      sides={axis === "y" ? location.X_LOCATIONS : location.Y_LOCATIONS}
    />
  </>
);

/**
 * Colors of the scale and its labels, and the text size. The fill color is the symbol's
 * own, so the caller renders it against whichever path holds it.
 */
export const StyleFields = ({ path }: FormProps): ReactElement => (
  <>
    <NodeForm.ColorField path={field(path, "axisColor")} label="Scale color" />
    <NodeForm.ColorField path={field(path, "textColor")} label="Text color" />
    <Base.Field<text.Level>
      path={field(path, "level")}
      label="Text size"
      padHelpText={false}
    >
      {NodeForm.SelectTextLevel}
    </Base.Field>
  </>
);
