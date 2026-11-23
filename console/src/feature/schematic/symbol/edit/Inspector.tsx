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
  const value =
    at == null
      ? Spec.own({ looks }, part, field.field)
      : Spec.effective({ looks }, part, field.field, at);
  return {
    ...inspector,
    ...field,
    value: value as Look[F] | undefined,
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
// field varies by state.
const Property = ({ field, label, icon, children }: PropertyProps): ReactElement => {
  const { part, values } = useInspectorContext("Property");
  const looks = Form.useFieldValue(path.data.looks);
  const { get, set } = Form.useContext();
  const [split, setSplit] = useState(false);
  const several = values.values.length > 1;
  const varies = several && (split || Spec.variesByValue({ looks }, part, field));
  const handleVariesChange = () => {
    const keys = values.values.map((v) => v.value);
    set(
      path.data,
      Spec.setVariesByValue(get(path.data).value, part, field, keys, !varies),
    );
    setSplit(!varies);
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
          {several && (
            <Button.Button
              size="small"
              variant={varies ? "filled" : "outlined"}
              onClick={handleVariesChange}
              tooltip={
                varies
                  ? `Use one ${name} in every state`
                  : `Set the ${name} of each state`
              }
            >
              By state
            </Button.Button>
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
          <Text.Text level="h5" weight={500}>
            {part?.name ?? "Whole drawing"}
          </Text.Text>
          <Text.Text level="small" color={8}>
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
