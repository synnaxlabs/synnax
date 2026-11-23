// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/feature/schematic/symbol/edit/Edit.css";

import type { schematic } from "@synnaxlabs/client";
import { Button } from "@synnaxlabs/lyra/button";
import { context } from "@synnaxlabs/lyra/context";
import { Divider } from "@synnaxlabs/lyra/divider";
import { Flex } from "@synnaxlabs/lyra/flex";
import { Form } from "@synnaxlabs/lyra/form";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Text } from "@synnaxlabs/lyra/text";
import { Schematic } from "@synnaxlabs/pluto";
import type { color } from "@synnaxlabs/x";
import { type PropsWithChildren, type ReactElement, useState } from "react";

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
  /** shown is the value whose looks the control shows. */
  shown: number;
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
  const { part, preview } = inspector;
  const { at } = field;
  // A control for every value shows what the canvas draws at the preview value.
  const shown = at ?? preview;
  const f = field.field as F;
  return {
    ...inspector,
    ...field,
    shown,
    value: Spec.effective({ looks }, part, f, shown),
    inherited:
      part == null ? undefined : Spec.effective({ looks }, undefined, f, shown),
    own: Spec.own({ looks }, part, f, shown),
    onChange: (v) => {
      const spec = get(path.data).value;
      // A stored static symbol can hold looks on Base, which an edit at every value
      // would not reach.
      const from =
        spec.variant === "static"
          ? { ...spec, looks: Spec.staticLooks(spec.looks) }
          : spec;
      set(path.data, Spec.setOwn(from, part, f, v, at));
    },
  };
};

interface PropertyProps extends PropsWithChildren {
  field: Spec.Field;
  label: string;
  icon: ReactElement;
}

// Property shows its control once for every value, or once for each value when the
// field varies by state. A symbol with one value has one control.
const Property = ({ field, label, icon, children }: PropertyProps): ReactElement => {
  const { part, values } = useInspectorContext("Property");
  const looks = Form.useFieldValue(path.data.looks);
  const { get, set } = Form.useContext();
  const [split, setSplit] = useState(() => Spec.variesByValue({ looks }, part, field));
  const several = values.list.length > 1;
  // A static symbol has one value, so it ends By state.
  if (split && !several) setSplit(false);
  const varies = several && (split || Spec.variesByValue({ looks }, part, field));
  const handleVariesChange = (next: boolean) => {
    set(
      path.data,
      Spec.setVariesByValue(
        get(path.data).value,
        part,
        field,
        values.list.map((v) => v.value),
        next,
      ),
    );
    setSplit(next);
  };
  const name = label.toLowerCase();
  return (
    <Flex.Box y gap="small">
      <Flex.Box x align="center" justify="between">
        <Text.Text level="small" gap="small">
          {icon}
          {label}
        </Text.Text>
        <Flex.Box x align="center" gap="small">
          {!varies && <FieldContext value={{ field, label }}>{children}</FieldContext>}
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
  const { value, inherited, own, onChange, label, shown, values } = useScope<
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
      entryColor={values.entryColor(shown)}
    />
  );
};

export interface InspectorProps {
  part: Spec.Target;
  /** preview is the value that the canvas draws. */
  preview: number;
}

/** Inspector edits the looks of one part, or of the whole drawing. */
export const Inspector = ({ part: key, preview }: InspectorProps): ReactElement => {
  const parts = Form.useFieldValue(path.data.parts);
  const markup = Form.useFieldValue(path.data.drawing.markup);
  const values = useValues();
  const part = parts.find((p) => p.key === key);
  const drawn =
    part == null ? undefined : Schematic.Node.Region.paint(markup, part.selectors);
  return (
    <InspectorContext value={{ part: part?.key, preview, values }}>
      <Flex.Box y gap="medium" className={CSS.B("schematic-inspector")}>
        <Flex.Box y gap={0}>
          <Text.Text level="h5">{part?.name ?? "Whole drawing"}</Text.Text>
          <Text.Text level="small" color={9}>
            {part == null
              ? "Looks here reach every shape."
              : "Click shapes in the drawing to add or remove them."}
          </Text.Text>
        </Flex.Box>
        <Divider.Divider x />
        <Property field="fill" label="Fill" icon={<Icon.FillColor />}>
          <PaintControl drawingColor={drawn?.fill} />
        </Property>
        <Property field="stroke" label="Stroke" icon={<Icon.StrokeColor />}>
          <PaintControl drawingColor={drawn?.stroke} />
        </Property>
      </Flex.Box>
    </InspectorContext>
  );
};
