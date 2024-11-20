// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type bounds, deep, scale } from "@synnaxlabs/x";
import { type ReactElement, useCallback } from "react";

import { Align } from "@/align";
import { Color } from "@/color";
import { Form } from "@/form";
import { Select } from "@/select";
import { Tabs } from "@/tabs";
import { Text } from "@/text";
import { Value } from "@/vis/value";

export const ValueForm = () => {
  const content: Tabs.RenderProp = useCallback(({ tabKey }) => {
    switch (tabKey) {
      case "telem":
        return (
          <Align.Space direction="y" style={{ padding: "2rem" }}>
            <Value.TelemForm path="" />
          </Align.Space>
        );
      case "redline":
        return (
          <Align.Space direction="y" style={{ padding: "2rem" }}>
            <RedlineForm />
          </Align.Space>
        );
      default:
        return (
          <Align.Space direction="y" grow empty style={{ padding: "2rem" }}>
            <Align.Space direction="x">
              <Form.Field<Color.Crude>
                hideIfNull
                label="Color"
                align="start"
                padHelpText={false}
                path="color"
              >
                {({ value, onChange, variant: _, ...props }) => (
                  <Color.Swatch
                    value={value ?? Color.ZERO.setAlpha(1).rgba255}
                    onChange={(v) => onChange(v.rgba255)}
                    {...props}
                    bordered
                  />
                )}
              </Form.Field>
              <Form.Field<Text.Level>
                path="level"
                label="Size"
                hideIfNull
                padHelpText={false}
              >
                {(p) => <Text.SelectLevel {...p} />}
              </Form.Field>
            </Align.Space>
          </Align.Space>
        );
    }
  }, []);
  const tabsProps = Tabs.useStatic({
    tabs: [
      { tabKey: "style", name: "Style" },
      { tabKey: "telem", name: "Telemetry" },
      { tabKey: "redline", name: "Redline" },
    ],
    content,
  });
  return <Tabs.Tabs {...tabsProps} />;
};

const RedlineForm = (): ReactElement => {
  const { set, get } = Form.useContext();
  const b = Form.useFieldValue<bounds.Bounds>("redline.bounds");
  const s = scale.Scale.scale<number>(0, 1).scale(b);
  return (
    <Align.Space direction="x" grow>
      <Form.NumericField
        inputProps={{ size: "small", showDragHandle: false }}
        style={{ width: 60 }}
        label="Lower"
        path="redline.bounds.lower"
      />
      <Form.Field<Color.Gradient>
        path="redline.gradient"
        label="Gradient"
        align="start"
        padHelpText={false}
      >
        {({ value, onChange }) => (
          <Color.GradientPicker
            value={deep.copy(value)}
            scale={s}
            onChange={(v) => {
              const prevB = get<bounds.Bounds>("redline.bounds").value;
              const nextBounds = { ...prevB };
              const positions = v.map((c) => c.position);
              const highestPos = s.pos(Math.max(...positions));
              const lowestPos = s.pos(Math.min(...positions));
              const highestGreater = highestPos > nextBounds.upper;
              const lowestLower = lowestPos < nextBounds.lower;
              if (highestGreater) {
                v[v.length - 1].position = 1;
                nextBounds.upper = highestPos;
              }
              if (lowestLower) {
                v[0].position = 0;
                nextBounds.lower = lowestPos;
              }
              const grad = v.map((c) => ({
                ...c,
                color: new Color.Color(c.color).hex,
              }));
              if (highestGreater || lowestLower)
                set("redline", {
                  bounds: nextBounds,
                  gradient: grad,
                });
              else
                onChange(v.map((c) => ({ ...c, color: new Color.Color(c.color).hex })));
            }}
          />
        )}
      </Form.Field>
      <Form.NumericField
        inputProps={{ size: "small", showDragHandle: false }}
        style={{ width: 60 }}
        label="Upper"
        path="redline.bounds.upper"
      />
    </Align.Space>
  );
};

export const TextForm = () => (
  <Align.Space direction="x" grow style={{ padding: "2rem" }}>
    <Form.TextField path="value" label="Text" />
    <Form.Field<Text.Level> path="level" label="Size" hideIfNull padHelpText={false}>
      {(p) => <Text.SelectLevel {...p} />}
    </Form.Field>
    <Form.Field<Text.Weight> path="weight" label="Weight" padHelpText={false}>
      {(p) => <Text.SelectWeight {...p} />}
    </Form.Field>
    <Form.Field<Align.Alignment> path="align" label="Alignment" hideIfNull>
      {(p) => <Select.TextAlignment {...p} />}
    </Form.Field>
    <Form.Field<Color.Crude>
      path="backgroundColor"
      label="Background"
      align="start"
      padHelpText={false}
    >
      {({ value, onChange }) => <Color.Swatch value={value} onChange={onChange} />}
    </Form.Field>
  </Align.Space>
);
