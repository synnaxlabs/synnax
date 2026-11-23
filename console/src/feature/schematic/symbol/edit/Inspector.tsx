// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/feature/schematic/symbol/edit/Edit.css";

import { schematic } from "@synnaxlabs/client";
import { Button } from "@synnaxlabs/lyra/button";
import { context } from "@synnaxlabs/lyra/context";
import { Divider } from "@synnaxlabs/lyra/divider";
import { Flex } from "@synnaxlabs/lyra/flex";
import { Form } from "@synnaxlabs/lyra/form";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Input } from "@synnaxlabs/lyra/input";
import { Select } from "@synnaxlabs/lyra/select";
import { Status } from "@synnaxlabs/lyra/status";
import { Text } from "@synnaxlabs/lyra/text";
import { Triggers } from "@synnaxlabs/lyra/triggers";
import { Schematic } from "@synnaxlabs/pluto";
import { type bounds, type color, TimeSpan } from "@synnaxlabs/x";
import { Fragment, type PropsWithChildren, type ReactElement, useState } from "react";

import { parentGroup } from "@/feature/schematic/symbol/edit/group";
import { Paint } from "@/feature/schematic/symbol/edit/Paint";
import { useValues, type Values } from "@/feature/schematic/symbol/edit/useValues";
import { Spec } from "@/feature/schematic/symbol/spec";
import { CSS } from "@/platform/css";

const path = Schematic.Symbol.formPath;
type Look = schematic.symbol.Look;

interface InspectorContextValue {
  part: Spec.Target;
  preview: number;
  values: Values;
}

const [InspectorContext, useInspectorContext] = context.create<InspectorContextValue>({
  displayName: "Symbol.Edit.InspectorContext",
  providerName: "Symbol.Edit.Inspector",
});

interface FieldContextValue {
  field: Spec.Field;
  label: string;
  /** at is the value that the control edits, or undefined for every value. */
  at?: number;
}

const [FieldContext, useFieldContext] = context.create<FieldContextValue>({
  displayName: "Symbol.Edit.FieldContext",
  providerName: "Symbol.Edit.Property",
});

interface Scope<F extends Spec.Field> extends InspectorContextValue, FieldContextValue {
  value: Look[F] | undefined;
  onChange: (value: Look[F] | undefined) => void;
}

// useScope returns the field that a control edits, at its value or at every value.
const useScope = <F extends Spec.Field>(name: string): Scope<F> => {
  const inspector = useInspectorContext(name);
  const field = useFieldContext(name);
  const looks = Form.useFieldValue(path.data.looks);
  const { get, set } = Form.useContext();
  const { part } = inspector;
  const { at } = field;
  return {
    ...inspector,
    ...field,
    value: Spec.effective({ looks }, part, field.field, at) as Look[F] | undefined,
    onChange: (v) =>
      set(path.data, Spec.setOwn(get(path.data).value, part, field.field, v, at)),
  };
};

interface PropertyProps extends PropsWithChildren {
  field: Spec.Field;
  label: string;
  icon: ReactElement;
}

// Property shows its control once for every value, or once for each value when the
// field varies by state. A symbol with one value shows its looks on that value too.
const Property = ({ field, label, icon, children }: PropertyProps): ReactElement => {
  const { part, values } = useInspectorContext("Property");
  const looks = Form.useFieldValue(path.data.looks);
  const { get, set } = Form.useContext();
  const [split, setSplit] = useState(false);
  const several = values.values.length > 1;
  const varies = split || Spec.variesByValue({ looks }, part, field);
  const handleVariesChange = (next: boolean) => {
    const keys = values.values.map((v) => v.value);
    set(
      path.data,
      Spec.setVariesByValue(get(path.data).value, part, field, keys, next),
    );
    setSplit(next);
  };
  const name = label.toLowerCase();
  return (
    <Flex.Box y gap="small">
      <Flex.Box x align="center" justify="between">
        <Text.Text level="small" weight={500} gap="small">
          {icon}
          {label}
        </Text.Text>
        <Flex.Box x align="center" gap="small">
          {!varies && <FieldContext value={{ field, label }}>{children}</FieldContext>}
          {(several || varies) && (
            <Button.Toggle
              size="small"
              value={varies}
              onChange={handleVariesChange}
              tooltip={
                varies
                  ? `Use one ${name} in every state`
                  : `Set the ${name} of each state`
              }
            >
              By state
            </Button.Toggle>
          )}
        </Flex.Box>
      </Flex.Box>
      {varies &&
        values.values.map((v) => (
          <Flex.Box
            key={v.key}
            x
            align="center"
            justify="between"
            className={CSS.BE("schematic-inspector", "value")}
          >
            <Text.Text level="small" color={9}>
              {v.name}
            </Text.Text>
            <FieldContext
              value={{ field, label: `${label} while ${v.name}`, at: v.value }}
            >
              {children}
            </FieldContext>
          </Flex.Box>
        ))}
    </Flex.Box>
  );
};

interface PaintControlProps {
  drawingColor?: color.Crude;
}

const PaintControl = ({ drawingColor }: PaintControlProps): ReactElement => {
  const { value, onChange, label, at, preview, values } = useScope<"fill" | "stroke">(
    "PaintControl",
  );
  return (
    <Paint
      aria-label={label}
      value={value}
      onChange={onChange}
      entryState={values.entryState}
      drawingColor={drawingColor}
      entryColor={values.entryColor(at ?? preview)}
    />
  );
};

const VisibleControl = (): ReactElement => {
  const { value, onChange, label, at } = useScope<"hidden">("VisibleControl");
  return (
    <Input.Checkbox
      aria-label={label}
      value={value !== true}
      onChange={(shown) => onChange(shown ? (at == null ? undefined : false) : true)}
    />
  );
};

const BLINK: schematic.symbol.Blink = { period: TimeSpan.SECOND.nanoseconds };

const BlinkControl = (): ReactElement => {
  const { value, onChange, label } = useScope<"blink">("BlinkControl");
  return (
    <Input.Checkbox
      aria-label={label}
      value={value != null}
      onChange={(on) => onChange(on ? BLINK : undefined)}
    />
  );
};

type MotionKind = schematic.symbol.Motion["type"];
type Loop = Extract<MotionKind, "spin" | "flow">;

interface MotionGroup {
  name: string;
  kinds: { kind: MotionKind; name: string }[];
}

// MOTIONS groups the kinds by how they move. The picker offers only the loops, since
// a motion that follows a value needs a number state.
const MOTIONS: MotionGroup[] = [
  {
    name: "Follow a value",
    kinds: [
      { kind: "level", name: "Fill" },
      { kind: "rotate", name: "Rotation" },
      { kind: "move", name: "Position" },
    ],
  },
  {
    name: "Loop",
    kinds: [
      { kind: "spin", name: "Spin" },
      { kind: "flow", name: "Flow" },
    ],
  },
];

const isLoop = (kind: MotionKind): kind is Loop => kind === "spin" || kind === "flow";

const RATE_BOUNDS: bounds.Bounds = { lower: 0, upper: Infinity };

const MotionControl = (): ReactElement => {
  const { value, onChange, label } = useScope<"motion">("MotionControl");
  const moving = value?.type === "spin" || value?.type === "flow" ? value : undefined;
  const handleKindChange = (kind: Loop | "none") => {
    if (kind === "none") return onChange(undefined);
    onChange(
      moving == null
        ? schematic.symbol.motionZ.parse({ type: kind })
        : { ...moving, type: kind },
    );
  };
  return (
    <Flex.Box y align="end" gap="small">
      <Select.Buttons<Loop | "none">
        value={value == null ? "none" : moving?.type}
        onChange={handleKindChange}
        size="small"
        align="center"
      >
        <Select.Item itemKey="none" size="small">
          None
        </Select.Item>
        {MOTIONS.map(({ name, kinds }) => {
          const offered = kinds.filter((k): k is { kind: Loop; name: string } =>
            isLoop(k.kind),
          );
          if (offered.length === 0) return null;
          return (
            <Fragment key={name}>
              <Divider.Divider y />
              <Text.Text level="small" color={8}>
                {name}
              </Text.Text>
              {offered.map((k) => (
                <Select.Item key={k.kind} itemKey={k.kind} size="small">
                  {k.name}
                </Select.Item>
              ))}
            </Fragment>
          );
        })}
      </Select.Buttons>
      {moving != null && (
        <Input.Numeric
          aria-label={`${label} rate`}
          size="small"
          value={moving.rate}
          onChange={(rate) => onChange({ ...moving, rate })}
          bounds={RATE_BOUNDS}
          showDragHandle={false}
          endContent="/s"
          className={CSS.BE("schematic-inspector", "rate")}
        />
      )}
    </Flex.Box>
  );
};

export interface InspectorProps {
  part: Spec.Target;
  /** preview is the value that the canvas draws. */
  preview: number;
}

const LEVEL_ISSUES: Record<Spec.LevelIssue, { message: string; description: string }> =
  {
    shapes: {
      message: "Fill motion needs one shape",
      description:
        "Fill motion fills each shape in this part by its own box. Use Select parent group to make the group around the shapes the part. If the shapes sit in no group, group them in the drawing, or remove the motion from the part to save.",
    },
    clip: {
      message: "Fill motion replaces a clip-path",
      description:
        "A shape in this part has its own clip-path, which fill motion would replace. Remove the clip-path from the drawing or the motion from the part to save.",
    },
  };

const SELECT_PARENT_TOOLTIP = (
  <Triggers.Text trigger={["Alt", "MouseLeft"]} level="small">
    Select parent group
  </Triggers.Text>
);

interface SelectParentProps {
  part: schematic.symbol.Part;
}

// SelectParent makes the group around the shapes of part the part.
const SelectParent = ({ part }: SelectParentProps): ReactElement | null => {
  const { get, set } = Form.useContext();
  const drawing = Form.useFieldValue(path.data.drawing);
  const group = parentGroup(drawing.markup, part.selectors);
  if (group == null) return null;
  const handleClick = () => {
    if (group.markup !== drawing.markup) set(path.data.drawing.markup, group.markup);
    const spec = {
      drawing: { ...drawing, markup: group.markup },
      parts: get(path.data.parts).value,
    };
    set(path.data.parts, Spec.toggleSelector(spec, part.key, group.selector));
  };
  return (
    <Button.Button
      size="small"
      variant="outlined"
      tooltip={SELECT_PARENT_TOOLTIP}
      onClick={handleClick}
    >
      Select parent group
    </Button.Button>
  );
};

/** Inspector edits the looks of one part, or of the whole drawing. */
export const Inspector = ({ part: key, preview }: InspectorProps): ReactElement => {
  const parts = Form.useFieldValue(path.data.parts);
  const drawing = Form.useFieldValue(path.data.drawing);
  const looks = Form.useFieldValue(path.data.looks);
  const values = useValues();
  const part = parts.find((p) => p.key === key);
  const drawn =
    part == null
      ? undefined
      : Schematic.Node.Region.paint(drawing.markup, part.selectors);
  const issue =
    part == null
      ? undefined
      : Spec.levelIssues({ drawing, parts, looks }).get(part.key);
  return (
    <InspectorContext value={{ part: part?.key, preview, values }}>
      <Flex.Box y gap="medium" className={CSS.B("schematic-inspector")}>
        <Flex.Box y gap={0}>
          <Text.Text level="h5" weight={500}>
            {part?.name ?? "Whole drawing"}
          </Text.Text>
          <Text.Text level="small" color={8}>
            {part == null
              ? "Looks here reach every shape. Click a shape to select its part or make one."
              : "Click shapes in the drawing to add or remove them."}
          </Text.Text>
        </Flex.Box>
        {part != null && <SelectParent part={part} />}
        <Divider.Divider x />
        <Property field="fill" label="Fill" icon={<Icon.FillColor />}>
          <PaintControl drawingColor={drawn?.fill} />
        </Property>
        <Property field="stroke" label="Stroke" icon={<Icon.StrokeColor />}>
          <PaintControl drawingColor={drawn?.stroke} />
        </Property>
        <Divider.Divider x />
        <Property field="hidden" label="Visible" icon={<Icon.Visible />}>
          <VisibleControl />
        </Property>
        <Property field="blink" label="Blink" icon={<Icon.Dot />}>
          <BlinkControl />
        </Property>
        <Property field="motion" label="Motion" icon={<Icon.Rotate />}>
          <MotionControl />
        </Property>
        {issue != null && (
          <Status.Summary level="small" variant="error" {...LEVEL_ISSUES[issue]} />
        )}
      </Flex.Box>
    </InspectorContext>
  );
};
