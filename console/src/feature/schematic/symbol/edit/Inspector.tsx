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

import { Paint } from "@/feature/schematic/symbol/edit/Paint";
import {
  useEntryColor,
  useValues,
  type Values,
} from "@/feature/schematic/symbol/edit/useValues";
import { Spec } from "@/feature/schematic/symbol/spec";
import { FLAGS } from "@/flags";
import { CSS } from "@/platform/css";

const path = Schematic.Symbol.formPath;
type Look = schematic.symbol.Look;

interface InspectorContextValue {
  part: Spec.Target;
  /** preview is the state that the inspector edits, at the value on the canvas. */
  preview: Spec.At;
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
  /** shown is the value of a state whose looks the control shows. */
  shown: Spec.At;
  looks: Look[];
  states: schematic.symbol.State[];
  value: Look[F] | undefined;
  /** inherited is what a look of the whole drawing, or of another state, gives. */
  inherited: Look[F] | undefined;
  /** inheritedFrom names the other state whose look gives inherited, if one does. */
  inheritedFrom?: string;
  /** own is what a look of the target with no condition or on the shown state gives. */
  own: Look[F] | undefined;
  /** lock says why the control cannot change its field, if it cannot. */
  lock?: string;
  onChange: (value: Look[F] | undefined) => void;
}

// useScope returns the field that a control edits, at its value or at every value.
const useScope = <F extends Spec.Field>(name: string): Scope<F> => {
  const inspector = useInspectorContext(name);
  const field = useFieldContext(name);
  const looks = Form.useFieldValue(path.data.looks);
  const states = Form.useFieldValue(path.data.states);
  const { get, set } = Form.useContext();
  const { part, preview } = inspector;
  const { at } = field;
  // A control for every value shows what the canvas draws at the preview value.
  const shown: Spec.At = at == null ? preview : { ...preview, value: at };
  const f = field.field as F;
  const symbol = { looks, states };
  const source = Spec.source(symbol, part, f, shown);
  const inherited = Spec.inheritedSource(symbol, part, f, shown);
  const other = (look?: Look): string | undefined => {
    const key = look?.when?.state;
    return key == null || key === shown.state
      ? undefined
      : states.find((s) => s.key === key)?.name;
  };
  // An edit at every value adds a look that a look of the target on a value hides.
  const covering = at == null && source?.part === part ? other(source) : undefined;
  // On a state with one value, the edit goes in at that value, after the covering look.
  const { list } = inspector.values;
  const one = covering != null && list.length === 1 ? list[0].value : undefined;
  const edited = at ?? one;
  // Property offers By state only for a state with more than one value.
  const orByState = list.length > 1 ? ", or set it by state" : "";
  return {
    ...inspector,
    ...field,
    shown,
    looks,
    states,
    value: source?.[f],
    inherited: inherited?.[f],
    inheritedFrom: other(inherited),
    own: Spec.own(symbol, part, f, shown),
    lock:
      covering == null || one != null
        ? undefined
        : `${covering} gives this. Change it there${orByState}.`,
    onChange: (v) => {
      const spec = get(path.data).value;
      const to = edited == null ? undefined : { ...shown, value: edited };
      set(path.data, Spec.setOwn(spec, part, f, v, to));
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
  const several = values.list.length > 1;
  // A symbol with no states has one value, so it ends By state.
  if (split && !several) setSplit(false);
  const varies =
    several && (split || Spec.variesByValue({ looks }, part, field, state));
  const handleVariesChange = (next: boolean) => {
    set(
      path.data,
      Spec.setVariesByValue(
        get(path.data).value,
        part,
        field,
        state,
        values.list.map((v) => v.value),
        next,
      ),
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
              aria-label={`${label} by state`}
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
        values.list.map((v) => (
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
  const { value, inherited, inheritedFrom, own, lock, onChange, label, shown, values } =
    useScope<"fill" | "stroke">("PaintControl");
  // An entry paint stores its own state, which can differ from the state shown.
  const entryColor = useEntryColor(
    value?.type === "entry"
      ? values.states.find((s) => s.key === value.state)
      : undefined,
    Spec.stateValues(shown),
  );
  return (
    <Paint
      aria-label={label}
      value={value}
      inherited={inherited}
      inheritedFrom={inheritedFrom}
      own={own}
      lock={lock}
      onChange={onChange}
      entryState={values.entryState}
      drawingColor={drawingColor}
      entryColor={entryColor}
    />
  );
};

const VisibleControl = (): ReactElement => {
  const {
    value,
    inherited,
    part,
    at,
    values,
    preview,
    looks,
    states,
    lock,
    onChange,
    label,
  } = useScope<"hidden">("VisibleControl");
  // Showing stores false when an inherited look hides the target here, or when the
  // drawing hides a part shown at every value at any value.
  const drawingHides =
    inherited === true ||
    (part != null &&
      at == null &&
      values.list.some(
        (v) =>
          Spec.effective({ looks, states }, undefined, "hidden", {
            ...preview,
            value: v.value,
          }) === true,
      ));
  return (
    <Input.Checkbox
      aria-label={label}
      value={value !== true}
      onChange={(shown) => onChange(shown ? (drawingHides ? false : undefined) : true)}
      disabled={lock != null}
      tooltip={lock}
    />
  );
};

const BLINK: schematic.symbol.Blink = { period: TimeSpan.SECOND.nanoseconds };

const BlinkControl = (): ReactElement => {
  const { value, inherited, inheritedFrom, own, lock, onChange, label } =
    useScope<"blink">("BlinkControl");
  // The whole drawing blinks as one shape, so a part can stop only its own blink.
  const locked = inherited != null && own == null;
  const why =
    lock ??
    (!locked
      ? undefined
      : inheritedFrom == null
        ? "The whole drawing blinks, and this part with it."
        : `${inheritedFrom} blinks this part.`);
  return (
    <Input.Checkbox
      aria-label={label}
      value={value != null}
      onChange={(on) => onChange(on ? BLINK : undefined)}
      disabled={why != null}
      tooltip={why}
    />
  );
};

type Loop = Extract<schematic.symbol.MotionType, "spin" | "flow">;

interface MotionGroup {
  name: string;
  kinds: { kind: schematic.symbol.MotionType; name: string }[];
}

// MOTIONS groups the kinds by how they move. The picker offers only the loops, since a
// motion that follows a value needs a number state.
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

const isLoop = (kind: schematic.symbol.MotionType): kind is Loop =>
  kind === "spin" || kind === "flow";

// The picker sets only loops. It shows a motion that follows a value, so a pick
// replaces it in sight.
const FOLLOWS = "Only a client can set a motion that follows a value.";

// A part under a motion of the whole drawing takes no new motion of its own.
const MOVES_WITH_DRAWING = "The whole drawing moves, and this part with it.";

const MotionControl = (): ReactElement => {
  const { value, inherited, inheritedFrom, own, lock, onChange, label } =
    useScope<"motion">("MotionControl");
  const locked = inherited != null && own == null;
  const why =
    lock ??
    (!locked
      ? undefined
      : inheritedFrom == null
        ? MOVES_WITH_DRAWING
        : `${inheritedFrom} moves this part.`);
  const moving = value?.type === "spin" || value?.type === "flow" ? value : undefined;
  const handleKindChange = (kind: schematic.symbol.MotionType | "none") => {
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
      <Select.Buttons<schematic.symbol.MotionType | "none">
        aria-label={label}
        value={value?.type ?? "none"}
        onChange={handleKindChange}
        size="small"
        align="center"
      >
        <Select.Item itemKey="none" size="small" disabled={why != null} tooltip={why}>
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
                  disabled={why != null || !isLoop(k.kind)}
                  tooltip={!isLoop(k.kind) ? FOLLOWS : why}
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
          disabled={why != null}
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
  /** preview is the state and value that the canvas draws, from Spec.resolvePreview. */
  preview: Spec.At;
}

const LEVEL_ISSUES: Record<
  Spec.LevelIssue,
  Pick<Status.SummaryProps, "message" | "description">
> = {
  shapes: {
    message: "Fill motion needs one shape or group",
    description:
      "Fill motion fills each shape or group in this part by its own box. Use Select parent group to make the group around the shapes the part. If the shapes sit in no group, group them in the drawing, or remove the motion from the part to save.",
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

const SelectParent = ({ part }: SelectParentProps): ReactElement => {
  const { get, set } = Form.useContext();
  const drawing = Form.useFieldValue(path.data.drawing);
  const group = Spec.parentGroup({ drawing }, part.selectors);
  const handleClick = () => {
    if (group == null) return;
    const { markup } = drawing;
    const named = Schematic.Node.Region.name(
      markup,
      group.ownerDocument.documentElement,
      group,
    );
    if (named.markup !== markup) set(path.data.drawing.markup, named.markup);
    set(
      path.data.parts,
      Spec.toggleSelector(get(path.data).value, part.key, named.selector),
    );
  };
  return (
    <Button.Button
      size="small"
      variant="outlined"
      tooltip={SELECT_PARENT_TOOLTIP}
      disabled={group == null}
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
    <InspectorContext
      value={{
        part: part?.key,
        preview,
        values,
      }}
    >
      <Flex.Box y gap="medium" className={CSS.B("schematic-inspector")}>
        <Flex.Box y gap={0}>
          <Text.Text level="h5">{part?.name ?? "Whole drawing"}</Text.Text>
          <Text.Text level="small" color={9}>
            {part == null
              ? "Looks here reach every shape. Click a shape to select its part or add one."
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
