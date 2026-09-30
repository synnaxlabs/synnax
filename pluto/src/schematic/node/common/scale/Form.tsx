// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type channel } from "@synnaxlabs/client";
import { Component } from "@synnaxlabs/lyra/component";
import { Form as Base } from "@synnaxlabs/lyra/form";
import { Input } from "@synnaxlabs/lyra/input";
import { Select } from "@synnaxlabs/lyra/select";
import {
  caseconv,
  type direction,
  location,
  type notation,
  primitive,
  type text,
} from "@synnaxlabs/x";
import { type PropsWithChildren, type ReactElement } from "react";

import { Channel } from "@/channel";
import { Notation } from "@/notation";
import { Form as NodeForm } from "@/schematic/node/common/form";
import { type Config } from "@/schematic/node/common/scale/config";
import { Scale as VisScale } from "@/vis/scale";
import { Staleness } from "@/vis/staleness";

const PRECISION_INPUT_PROPS: Partial<Input.NumericProps> = {
  bounds: { lower: 0, upper: 10 },
};
const WINDOW_SIZE_INPUT_PROPS: Partial<Input.NumericProps> = {
  bounds: { lower: 1, upper: 100 },
};

const NotationSelect = Component.renderProp(
  ({ value, onChange }: Input.Control<notation.Notation>): ReactElement => (
    <Notation.Select value={value} onChange={onChange} />
  ),
);

const SIDES: readonly location.Outer[] = [
  ...location.Y_LOCATIONS,
  ...location.X_LOCATIONS,
];

interface SideFieldProps {
  path: string;
  label: string;
  /** The sides to offer. A symbol with a fixed axis offers only the two it can use. */
  sides: readonly location.Outer[];
}

// A field on the other axis takes the side facing the same way as the default.
const SideField = ({ path, label, sides }: SideFieldProps): ReactElement => (
  <Base.Field<location.Outer> path={path} label={label} padHelpText={false}>
    {({ value, onChange }) => (
      <Select.Buttons value={value} onChange={onChange}>
        {sides.map((side) => (
          <Select.Item key={side} itemKey={side}>
            {caseconv.capitalize(side)}
          </Select.Item>
        ))}
      </Select.Buttons>
    )}
  </Base.Field>
);

export interface TelemFormProps {
  /** When true, clearing the channel unbinds the scale instead of pinning it to 0. */
  allowNone?: boolean;
}

/** TelemForm renders telemetry sections; place it inside `Form.Sections`. */
export const TelemForm = ({ allowNone = false }: TelemFormProps): ReactElement => {
  const { set } = Base.useContext();
  const channel = Base.useFieldValue<Config["channel"]>("channel", { optional: true });
  const handleChannelChange = (key: channel.Key | null): void => {
    if (allowNone && !primitive.isNonZero(key)) return set("channel", undefined);
    set("channel", key ?? 0);
  };
  return (
    <>
      <Base.Section title="Source">
        <Input.Item label="Channel" padHelpText={false}>
          <Channel.SelectSingle
            value={channel ?? 0}
            onChange={handleChannelChange}
            allowNone={allowNone}
          />
        </Input.Item>
        <Base.NumericField
          path="rollingAverage"
          label="Averaging window"
          padHelpText={false}
          inputProps={WINDOW_SIZE_INPUT_PROPS}
        />
      </Base.Section>
      <Base.Section title="Range">
        <NodeForm.BoundsFields path="bounds" padHelpText={false} />
      </Base.Section>
      <Base.Section title="Format">
        <Base.Field<notation.Notation>
          path="notation"
          label="Notation"
          padHelpText={false}
        >
          {NotationSelect}
        </Base.Field>
        <Base.NumericField
          path="precision"
          label="Precision"
          padHelpText={false}
          inputProps={PRECISION_INPUT_PROPS}
        />
        <NodeForm.UnitsField path="units" />
      </Base.Section>
      <Base.Section title="Staleness">
        <Staleness.Fields />
      </Base.Section>
    </>
  );
};

export interface DisplayFieldsProps extends PropsWithChildren {
  /** The axis the bar fills along, which the ticks must sit clear of. */
  axis?: direction.Direction;
}

/**
 * Which parts of the scale are drawn, and the sides the ticks and readout sit on. The
 * children are the value and scale switches, which each symbol stores its own way.
 */
export const DisplayFields = ({
  axis = "y",
  children,
}: DisplayFieldsProps): ReactElement => (
  <>
    <NodeForm.NegatedSwitchField path="levelHidden" label="Fill" padHelpText={false} />
    {children}
    <SideField path="caretSide" label="Value side" sides={SIDES} />
    <SideField
      path="side"
      label="Scale side"
      sides={axis === "y" ? location.X_LOCATIONS : location.Y_LOCATIONS}
    />
  </>
);

/** Colors of the level and the labels, and the text size. */
export const StyleFields = (): ReactElement => (
  <>
    <NodeForm.ColorField
      path="levelColor"
      label="Level"
      fallback={VisScale.levelColorFallback}
    />
    <NodeForm.ColorField
      path="textColor"
      label="Text"
      fallback={VisScale.textColorFallback}
    />
    <Base.Field<text.Level> path="level" label="Text size" padHelpText={false}>
      {NodeForm.SelectTextLevel}
    </Base.Field>
  </>
);
