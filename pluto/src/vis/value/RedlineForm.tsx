// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/vis/value/RedlineForm.css";

import { Button } from "@synnaxlabs/lyra/button";
import { CSS } from "@synnaxlabs/lyra/css";
import { Flex } from "@synnaxlabs/lyra/flex";
import { Form } from "@synnaxlabs/lyra/form";
import { Icon } from "@synnaxlabs/lyra/icon";
import { List } from "@synnaxlabs/lyra/list";
import { Select } from "@synnaxlabs/lyra/select";
import { Text } from "@synnaxlabs/lyra/text";
import { Theming } from "@synnaxlabs/lyra/theming";
import { color, type compare, id } from "@synnaxlabs/x";
import { type CSSProperties, type ReactElement } from "react";

import { Color } from "@/color";

const ascending: compare.Comparator<color.Band> = (a, b) => a.threshold - b.threshold;

interface BarProps {
  /** The CSS background the canvas paints for the band. */
  on: string;
  /** The CSS background during the off half of a flash, for a flashing band. */
  off?: string;
}

// One segment of the color bar down the left edge of the list. Adjacent segments join
// into one bar that paints what the canvas paints.
const Bar = ({ on, off }: BarProps): ReactElement => (
  <div
    className={CSS.cls(CSS.BE("redline-form", "bar"), off != null && CSS.M("flashing"))}
    style={{ "--pluto-redline-on": on, "--pluto-redline-off": off } as CSSProperties}
  />
);

interface BandItemProps {
  itemKey: string;
  index: number;
  path: string;
  units: string;
  presets: color.Crude[];
  bar: BarProps;
  duplicate: boolean;
  onRemove: (key: string) => void;
}

const BandItem = ({
  itemKey,
  index,
  path,
  units,
  presets,
  bar,
  duplicate,
  onRemove,
}: BandItemProps): ReactElement => {
  const bandPath = `${path}.${itemKey}`;
  return (
    <List.Item itemKey={itemKey} index={index} x align="center" gap="small">
      <Bar {...bar} />
      <Form.Field<color.Crude>
        path={`${bandPath}.color`}
        showLabel={false}
        showHelpText={false}
      >
        {({ value, onChange }) => (
          <Color.Swatch
            value={value}
            onChange={onChange}
            presets={presets}
            size="small"
            bordered
          />
        )}
      </Form.Field>
      <Text.Text level="small" color={9} className={CSS.BE("redline-form", "relation")}>
        ≥
      </Text.Text>
      <Form.NumericField
        path={`${bandPath}.threshold`}
        showLabel={false}
        showHelpText={false}
        grow
        inputProps={{
          size: "small",
          showDragHandle: false,
          units,
          status: duplicate ? "error" : undefined,
          tooltip: duplicate ? "Duplicate threshold" : undefined,
        }}
      />
      <Flex.Box x gap="small" className={CSS.BE("redline-form", "actions")}>
        <Form.Field<boolean>
          path={`${bandPath}.flashing`}
          showLabel={false}
          showHelpText={false}
        >
          {({ value, onChange }) => (
            <Button.Toggle
              value={value}
              onChange={onChange}
              size="small"
              tooltip="Flash while the value is in this band"
            >
              <Icon.Bolt />
            </Button.Toggle>
          )}
        </Form.Field>
        <Button.Button
          onClick={() => onRemove(itemKey)}
          size="small"
          variant="text"
          aria-label="Remove band"
          reveal
        >
          <Icon.Close />
        </Button.Button>
      </Flex.Box>
    </List.Item>
  );
};

interface FloorItemProps {
  background?: color.Color;
  /** The lowest band threshold, or undefined when there are no bands. */
  threshold?: number;
  units: string;
}

// Values below every threshold show the value's background, set in the style tab. The
// item mirrors a band's columns so the two line up.
const FloorItem = ({ background, threshold, units }: FloorItemProps): ReactElement => (
  <Flex.Box x align="center" gap="small" className={CSS.BE("redline-form", "floor")}>
    <Bar on={color.cssString(background ?? color.ZERO)} />
    <Color.Swatch
      value={background ?? color.ZERO}
      allowChange={false}
      size="small"
      bordered
      tooltip="Set the background in the style tab"
    />
    {threshold != null && (
      <Text.Text level="small" color={9} className={CSS.BE("redline-form", "relation")}>
        &lt;
      </Text.Text>
    )}
    <Text.Text
      level="small"
      grow
      square={false}
      className={CSS.BE("redline-form", "floor-value")}
    >
      {threshold == null ? "All values" : `${threshold} ${units}`.trimEnd()}
    </Text.Text>
    <Flex.Box x className={CSS.BE("redline-form", "actions")}>
      <Text.Text level="small" color={9}>
        {background == null ? "No fill" : "Background"}
      </Text.Text>
    </Flex.Box>
  </Flex.Box>
);

interface TransitionFieldProps {
  path: string;
}

const TransitionField = ({ path }: TransitionFieldProps): ReactElement => (
  <Form.Field<boolean> path={`${path}.smooth`} label="Transition" padHelpText={false}>
    {({ value, onChange }) => (
      <Select.Buttons
        value={value ? "smooth" : "steps"}
        onChange={(v: string) => onChange(v === "smooth")}
      >
        <Select.Item itemKey="steps">Steps</Select.Item>
        <Select.Item itemKey="smooth">Smooth</Select.Item>
      </Select.Buttons>
    )}
  </Form.Field>
);

export interface RedlineFormProps {
  /** The path of the redline. */
  path: string;
}

/**
 * Edits a redline as a list of threshold bands, lowest first. Rows sort by threshold
 * as thresholds change. Reads the value's units and background from the form root.
 */
export const RedlineForm = ({ path }: RedlineFormProps): ReactElement => {
  const bandsPath = `${path}.bands`;
  const redline = Form.useFieldValue<color.Scale>(path);
  const units = Form.useFieldValue<string>("units", { optional: true }) ?? "";
  const background =
    Form.useFieldValue<color.Color>("backgroundColor", { optional: true }) ?? undefined;
  const { push, remove } = Form.useFieldListUtils<string, color.Band>(bandsPath);
  const theme = Theming.use();
  // The form edits values in place, so values read here keep their identity across
  // edits and cannot key a memo.
  const { bands, smooth } = redline;
  const sorted = bands.toSorted(ascending);
  const keys = sorted.map(({ key }) => key);
  const backgroundCSS = color.cssString(background ?? color.ZERO);
  const items = new Map(
    sorted.map(({ key, threshold, color: c, flashing }, i) => {
      const css = color.cssString(c);
      const next = sorted.at(i + 1);
      const on =
        smooth && next != null
          ? `linear-gradient(to bottom, ${css}, ${color.cssString(next.color)})`
          : css;
      const bar: BarProps = { on, off: flashing ? backgroundCSS : undefined };
      const duplicate = sorted.some((b, j) => j !== i && b.threshold === threshold);
      return [key, { bar, duplicate }];
    }),
  );
  const presets = [
    theme.colors.secondary.z,
    theme.colors.warning.z,
    theme.colors.error.z,
    theme.colors.primary.z,
  ];
  const lowest = sorted.at(0);

  const handleAdd = (): void => {
    const [highest, second] = sorted.toReversed();
    let threshold = 0;
    if (highest != null) {
      const step = second == null ? 0 : highest.threshold - second.threshold;
      threshold = highest.threshold + (step > 0 ? step : 1);
    }
    const palette = [
      theme.colors.warning.z,
      theme.colors.error.z,
      ...(theme.colors.visualization.palettes.default ?? []),
    ];
    push(
      {
        key: id.create(),
        threshold,
        color: color.construct(palette[bands.length % palette.length]),
        flashing: false,
      },
      ascending,
    );
  };

  return (
    <Form.Sections x className={CSS.B("redline-form")}>
      <Form.Section title="Bands">
        <Flex.Box y empty className={CSS.BE("redline-form", "bands")}>
          <FloorItem
            background={background}
            threshold={lowest?.threshold}
            units={units}
          />
          <List.Frame data={keys}>
            <List.Scroll>
              <List.Items<string>>
                {({ itemKey, index }) => {
                  const item = items.get(itemKey);
                  if (item == null) return null;
                  return (
                    <BandItem
                      key={itemKey}
                      itemKey={itemKey}
                      index={index}
                      path={bandsPath}
                      units={units}
                      presets={presets}
                      onRemove={remove}
                      {...item}
                    />
                  );
                }}
              </List.Items>
            </List.Scroll>
          </List.Frame>
          <Button.Button
            onClick={handleAdd}
            variant="text"
            size="small"
            textColor={10}
            className={CSS.BE("redline-form", "add")}
          >
            <Icon.Add />
            Add band
          </Button.Button>
        </Flex.Box>
      </Form.Section>
      <Form.Section title="Options">
        <TransitionField path={path} />
      </Form.Section>
    </Form.Sections>
  );
};
