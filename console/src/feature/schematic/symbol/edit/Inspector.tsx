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
import { type color, TimeSpan } from "@synnaxlabs/x";
import { Fragment, type PropsWithChildren, type ReactElement, useState } from "react";

import { parentGroup } from "@/feature/schematic/symbol/edit/group";
import { Paint } from "@/feature/schematic/symbol/edit/Paint";
import { useValues, type Values } from "@/feature/schematic/symbol/edit/useValues";
import { Spec } from "@/feature/schematic/symbol/spec";
import { FLAGS } from "@/flags";
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
  /** inherited is what a look on the whole drawing gives the part. */
  inherited: Look[F] | undefined;
  /** own is what a look of the part itself gives it. */
  own: Look[F] | undefined;
  onChange: (value: Look[F] | undefined) => void;
}

// useScope returns the field that a control edits, at its value or at every value.
const useScope = <F extends Spec.Field>(name: string): Scope<F> => {
  const inspector = useInspectorContext(name);
  const field = useFieldContext(name);
  const looks = Form.useFieldValue(path.data.looks);
  const { get, set } = Form.useContext();
  const { part, preview, values } = inspector;
  const { at } = field;
  // A control for every value shows what the canvas draws at the preview value.
  const shown: Spec.At = { state: values.state, value: at ?? preview };
  return {
    ...inspector,
    ...field,
    value: Spec.effective({ looks }, part, field.field, shown) as Look[F] | undefined,
    inherited:
      part == null
        ? undefined
        : (Spec.effective({ looks }, undefined, field.field, shown) as
            Look[F] | undefined),
    own: Spec.effective(
      { looks: looks.filter((l) => l.part === part) },
      part,
      field.field,
      shown,
    ) as Look[F] | undefined,
    onChange: (v) => {
      const spec = get(path.data).value;
      // A stored static symbol can hold looks on Base, which an edit at every value
      // would not reach.
      const from =
        spec.variant === "static"
          ? { ...spec, looks: Spec.staticLooks(spec.looks) }
          : spec;
      const edited = at == null ? undefined : { state: values.state, value: at };
      set(path.data, Spec.setOwn(from, part, field.field, v, edited));
    },
  };
};

interface PropertyProps extends PropsWithChildren {
  field: Spec.Field;
  label: string;
  icon: ReactElement;
  /** wide puts the control for every value under the label. */
  wide?: boolean;
}

// Property shows its control once for every value, or once for each value when the
// field varies by state. A symbol with one value has one control.
const Property = ({
  field,
  label,
  icon,
  wide = false,
  children,
}: PropertyProps): ReactElement => {
  const { part, values } = useInspectorContext("Property");
  const looks = Form.useFieldValue(path.data.looks);
  const { get, set } = Form.useContext();
  const { state } = values;
  const [split, setSplit] = useState(() =>
    Spec.variesByValue({ looks }, part, field, state),
  );
  const several = values.values.length > 1;
  // A static symbol has one value, so it ends By state.
  if (split && !several) setSplit(false);
  const varies =
    several && (split || Spec.variesByValue({ looks }, part, field, state));
  const handleVariesChange = (next: boolean) => {
    const keys = values.values.map((v) => v.value);
    set(
      path.data,
      Spec.setVariesByValue(get(path.data).value, part, field, state, keys, next),
    );
    setSplit(next);
  };
  const name = label.toLowerCase();
  const control = varies ? null : (
    <FieldContext value={{ field, label }}>{children}</FieldContext>
  );
  return (
    <Flex.Box y gap="small">
      <Flex.Box x align="center" justify="between">
        <Text.Text level="small" gap="small">
          {icon}
          {label}
        </Text.Text>
        <Flex.Box x align="center" gap="small">
          {!wide && control}
          {several && (
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
      {wide && control}
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
  const { value, inherited, own, onChange, label, at, preview, values } = useScope<
    "fill" | "stroke"
  >("PaintControl");
  return (
    <Paint
      aria-label={label}
      value={value}
      inherited={inherited}
      own={own}
      onChange={onChange}
      entryState={values.entryState}
      drawingColor={drawingColor}
      entryColor={values.entryColor(at ?? preview)}
    />
  );
};

const VisibleControl = (): ReactElement => {
  const { value, inherited, part, at, values, onChange, label } =
    useScope<"hidden">("VisibleControl");
  const looks = Form.useFieldValue(path.data.looks);
  // A part shown at every value stores false when the drawing hides at any value.
  const drawingHides =
    part != null &&
    (at == null
      ? values.values.some(
          (v) =>
            Spec.effective({ looks }, undefined, "hidden", {
              state: values.state,
              value: v.value,
            }) === true,
        )
      : inherited === true);
  return (
    <Input.Checkbox
      aria-label={label}
      value={value !== true}
      onChange={(shown) => onChange(shown ? (drawingHides ? false : undefined) : true)}
    />
  );
};

const BLINK: schematic.symbol.Blink = { period: TimeSpan.SECOND.nanoseconds };

const BlinkControl = (): ReactElement => {
  const { value, inherited, own, onChange, label } = useScope<"blink">("BlinkControl");
  // The whole drawing blinks as one shape, so a part can stop only its own blink.
  const locked = inherited != null && own == null;
  return (
    <Input.Checkbox
      aria-label={label}
      value={value != null}
      onChange={(on) => onChange(on ? BLINK : undefined)}
      disabled={locked}
      tooltip={locked ? "The whole drawing blinks, and this part with it." : undefined}
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

// The picker sets only loops. It shows a motion that follows a value, so a pick
// replaces it in sight.
const FOLLOWS = "Only a client can set a motion that follows a value.";

const MotionControl = (): ReactElement => {
  const { value, inherited, own, onChange, label } =
    useScope<"motion">("MotionControl");
  const locked = inherited != null && own == null;
  const moving = value?.type === "spin" || value?.type === "flow" ? value : undefined;
  const handleKindChange = (kind: MotionKind | "none") => {
    if (kind === "none") return onChange(undefined);
    if (!isLoop(kind)) throw new Error(`the motion picker cannot set ${kind}`);
    onChange(
      moving == null
        ? schematic.symbol.motionZ.parse({ type: kind })
        : { ...moving, type: kind },
    );
  };
  return (
    <Flex.Box y align="end" gap="small">
      <Select.Buttons<MotionKind | "none">
        value={value?.type ?? "none"}
        onChange={handleKindChange}
        size="small"
        align="center"
      >
        <Select.Item
          itemKey="none"
          size="small"
          disabled={locked}
          tooltip={
            locked ? "The whole drawing moves, and this part with it." : undefined
          }
        >
          None
        </Select.Item>
        {MOTIONS.map(({ name, kinds }) => {
          const offered = kinds.filter((k) => isLoop(k.kind) || k.kind === value?.type);
          if (offered.length === 0) return null;
          return (
            <Fragment key={name}>
              <Divider.Divider y />
              <Text.Text level="small" color={8}>
                {name}
              </Text.Text>
              {offered.map((k) => (
                <Select.Item
                  key={k.kind}
                  itemKey={k.kind}
                  size="small"
                  disabled={!isLoop(k.kind)}
                  tooltip={isLoop(k.kind) ? undefined : FOLLOWS}
                >
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
          disabled={locked}
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
  /** preview is the value of a state that the canvas draws. */
  preview: Spec.At;
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
  const values = useValues(preview.state);
  const part = parts.find((p) => p.key === key);
  const drawn =
    part == null
      ? undefined
      : Schematic.Node.Region.paint(drawing.markup, part.selectors);
  const issue =
    FLAGS.symbolMotion && part != null
      ? Spec.levelIssues({ drawing, parts, looks }).get(part.key)
      : undefined;
  return (
    <InspectorContext value={{ part: part?.key, preview: preview.value, values }}>
      <Flex.Box y gap="medium" className={CSS.B("schematic-inspector")}>
        <Flex.Box y gap={0}>
          <Text.Text level="h5">{part?.name ?? "Whole drawing"}</Text.Text>
          <Text.Text level="small" color={9}>
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
        {FLAGS.symbolMotion && (
          <>
            <Property field="blink" label="Blink" icon={<Icon.Dot />}>
              <BlinkControl />
            </Property>
            <Property field="motion" label="Motion" icon={<Icon.Rotate />} wide>
              <MotionControl />
            </Property>
            {issue != null && (
              <Status.Summary level="small" variant="error" {...LEVEL_ISSUES[issue]} />
            )}
          </>
        )}
      </Flex.Box>
    </InspectorContext>
  );
};
