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
import { Text } from "@synnaxlabs/lyra/text";
import { Theming } from "@synnaxlabs/lyra/theming";
import { color, type compare, id } from "@synnaxlabs/x";
import { type CSSProperties, type ReactElement, useMemo } from "react";

import { Color } from "@/color";
import { type Band, type Redline } from "@/vis/value/redline";

const ascending: compare.Comparator<Band> = (a, b) => a.threshold - b.threshold;

// The share of the preview strip given to the base on each side of the thresholds.
const PREVIEW_PAD = 0.2;

const previewBackground = ({ bands, base, smooth }: Redline): string => {
  const sorted = bands.toSorted(ascending);
  const baseCSS = color.cssString(base ?? color.ZERO);
  if (sorted.length === 0) return baseCSS;
  const lowest = sorted[0].threshold;
  const span = sorted[sorted.length - 1].threshold - lowest || 1;
  const percent = (threshold: number): number =>
    ((threshold - lowest) / span + PREVIEW_PAD) * (100 / (1 + 2 * PREVIEW_PAD));
  const stops = [`${baseCSS} 0%`, `${baseCSS} ${percent(lowest)}%`];
  sorted.forEach(({ threshold, color: c }, i) => {
    const css = color.cssString(c);
    stops.push(`${css} ${percent(threshold)}%`);
    if (smooth) return;
    const next = sorted[i + 1];
    stops.push(`${css} ${next == null ? 100 : percent(next.threshold)}%`);
  });
  if (smooth) stops.push(`${color.cssString(sorted[sorted.length - 1].color)} 100%`);
  return `linear-gradient(to right, ${stops.join(", ")})`;
};

interface BandItemProps {
  itemKey: string;
  index: number;
  path: string;
  units: string;
  presets: color.Crude[];
  duplicate: boolean;
  onRemove: (key: string) => void;
}

const BandItem = ({
  itemKey,
  index,
  path,
  units,
  presets,
  duplicate,
  onRemove,
}: BandItemProps): ReactElement => {
  const bandPath = `${path}.${itemKey}`;
  return (
    <List.Item itemKey={itemKey} index={index} x align="center" gap="small">
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
    </List.Item>
  );
};

interface BaseItemProps {
  path: string;
  presets: color.Crude[];
}

// The base is optional, and absent paints nothing. Form.Field hides an absent path, so
// the swatch reads the value directly and writes only what the user picks.
const BaseItem = ({ path, presets }: BaseItemProps): ReactElement => {
  const { set } = Form.useContext();
  const basePath = `${path}.base`;
  const base = Form.useFieldValue<color.Color>(basePath, { optional: true });
  return (
    <Flex.Box x align="center" gap="small" grow>
      <Color.Swatch
        value={base ?? color.ZERO}
        onChange={(next: color.Color) => set(basePath, next)}
        presets={presets}
        size="small"
        bordered
      />
      <Text.Text level="small" status={base == null ? "disabled" : undefined} grow>
        {base == null ? "Base: no fill" : "Base"}
      </Text.Text>
      {base != null && (
        <Button.Button
          onClick={() => set(basePath, undefined)}
          size="small"
          variant="text"
          tooltip="Paint nothing below the lowest threshold"
        >
          <Icon.Close />
        </Button.Button>
      )}
    </Flex.Box>
  );
};

export interface RedlineFormProps {
  /** The path of the redline. Band thresholds show the units at the root `units`. */
  path: string;
}

/**
 * Edits a redline as a list of threshold bands, highest first, above a base for values
 * below every threshold. Rows sort by threshold as thresholds change.
 */
export const RedlineForm = ({ path }: RedlineFormProps): ReactElement => {
  const bandsPath = `${path}.bands`;
  const redline = Form.useFieldValue<Redline>(path);
  const units = Form.useFieldValue<string>("units", { optional: true }) ?? "";
  const { push, remove } = Form.useFieldListUtils<string, Band>(bandsPath);
  const theme = Theming.use();
  // The form edits values in place, so values read here keep their identity across
  // edits and cannot key a memo.
  const { bands } = redline;
  const keys = bands.toSorted((a, b) => ascending(b, a)).map(({ key }) => key);
  const seen = new Set<number>();
  const duplicates = new Set<number>();
  bands.forEach(({ threshold }) =>
    (seen.has(threshold) ? duplicates : seen).add(threshold),
  );
  const presets = useMemo(
    () => [
      theme.colors.secondary.z,
      theme.colors.warning.z,
      theme.colors.error.z,
      theme.colors.primary.z,
    ],
    [theme],
  );
  const preview: CSSProperties = { background: previewBackground(redline) };

  const handleAdd = (): void => {
    const sorted = bands.toSorted((a, b) => ascending(b, a));
    const [highest, second] = sorted;
    let threshold = 0;
    if (highest != null) {
      const step = second == null ? 0 : highest.threshold - second.threshold;
      threshold = highest.threshold + (step > 0 ? step : 1);
    }
    const seeds = [
      theme.colors.warning.z,
      theme.colors.error.z,
      ...(theme.colors.visualization.palettes.default ?? []),
    ];
    push(
      {
        key: id.create(),
        threshold,
        color: color.construct(seeds[bands.length % seeds.length]),
        flashing: false,
      },
      ascending,
    );
  };

  return (
    <Flex.Box x gap="large" grow className={CSS.B("redline-form")}>
      <Flex.Box y gap="small" className={CSS.BE("redline-form", "bands")}>
        <List.Frame data={keys}>
          <List.Scroll grow>
            <List.Items<string>
              emptyContent={
                <Flex.Box center grow>
                  <Text.Text y center status="disabled" gap="tiny">
                    No bands
                    <Text.Text variant="link" onClick={handleAdd}>
                      Add a band
                    </Text.Text>
                  </Text.Text>
                </Flex.Box>
              }
            >
              {({ itemKey, index }) => {
                const band = bands.find(({ key }) => key === itemKey);
                if (band == null) return null;
                return (
                  <BandItem
                    key={itemKey}
                    itemKey={itemKey}
                    index={index}
                    path={bandsPath}
                    units={units}
                    presets={presets}
                    duplicate={duplicates.has(band.threshold)}
                    onRemove={remove}
                  />
                );
              }}
            </List.Items>
          </List.Scroll>
        </List.Frame>
        <Flex.Box x align="center" className={CSS.BE("redline-form", "footer")}>
          <BaseItem path={path} presets={presets} />
          {bands.length > 0 && (
            <Button.Button
              onClick={handleAdd}
              variant="text"
              size="small"
              textColor={10}
            >
              <Icon.Add />
              Add band
            </Button.Button>
          )}
        </Flex.Box>
      </Flex.Box>
      <Flex.Box y gap="small" grow className={CSS.BE("redline-form", "display")}>
        <Form.SwitchField
          path={`${path}.smooth`}
          label="Smooth"
          padHelpText={false}
          x
          align="center"
        />
        <div className={CSS.BE("redline-form", "preview")} style={preview} />
      </Flex.Box>
    </Flex.Box>
  );
};
