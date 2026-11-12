// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type label, lineplot, ranger } from "@synnaxlabs/client";
import { Button } from "@synnaxlabs/lyra/button";
import { Color } from "@synnaxlabs/lyra/color";
import { Divider } from "@synnaxlabs/lyra/divider";
import { Flex } from "@synnaxlabs/lyra/flex";
import { Form } from "@synnaxlabs/lyra/form";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Theming } from "@synnaxlabs/lyra/theming";
import { Access, Label as PLabel, Ranger } from "@synnaxlabs/pluto";
import { type ReactElement } from "react";

import { Core } from "@/platform/core";
import { CSS } from "@/platform/css";
import { Framer } from "@/platform/framer";
import { Label } from "@/platform/label";
import { Range } from "@/platform/range";

const path = Ranger.formPath;

interface RangeKeyProps {
  rangeKey: ranger.Key;
}

const Crumbs = ({ rangeKey }: RangeKeyProps): ReactElement | null => {
  const range = Ranger.useResult({ key: rangeKey }).data;
  const open = Range.useOpen();
  if (range == null) return null;
  return (
    <>
      {range.parent != null && (
        <>
          <Crumbs rangeKey={range.parent} />
          <Icon.Caret.Right className={CSS.BE("range-overview", "separator")} />
        </>
      )}
      <Button.Button
        variant="text"
        size="small"
        textColor={9}
        gap="small"
        className={CSS.BE("range-overview", "crumb")}
        onClick={() => open(range.key)}
      >
        {range.name}
      </Button.Button>
    </>
  );
};

const Labels = ({ rangeKey, preview }: RangeKeyProps & { preview: boolean }) => {
  const value = Ranger.useResult({ key: rangeKey }).data?.labels ?? [];
  const { update } = PLabel.useEditFor();
  const handleChange = (next: label.Key[]): void =>
    update({
      id: ranger.resourceID(rangeKey),
      added: next.filter((key) => !value.includes(key)),
      removed: value.filter((key) => !next.includes(key)),
    });
  return (
    <Label.SelectMultiple
      zIndex={100}
      variant="floating"
      className={CSS.BE("range-overview", "labels-select")}
      value={value}
      onChange={handleChange}
      preview={preview}
    />
  );
};

interface SwatchProps {
  range: ranger.Range;
  preview: boolean;
}

const Swatch = ({ range, preview }: SwatchProps): ReactElement => {
  const { update } = Ranger.useSetColor();
  const { colors } = Theming.use();
  return (
    <Color.Swatch
      variant="shadow"
      value={range.color}
      fallback={colors.gray.l8}
      onChange={preview ? undefined : (color) => update({ key: range.key, color })}
      onlyChangeOnBlur
      tooltip="Color"
      className={CSS.BE("range-overview", "color")}
    />
  );
};

export interface HeaderProps {
  range: ranger.Range;
}

export const Header = ({ range }: HeaderProps): ReactElement => {
  const rangeKey = range.key;
  const canEdit = Access.useUpdateGranted(ranger.resourceID(rangeKey));
  const canPlot = Access.useCreateGranted(lineplot.TYPE_RESOURCE_ID);
  const parent = Ranger.useResultParent({ key: rangeKey }).data;
  const { form } = Ranger.useForm({
    query: { key: rangeKey },
    mode: canEdit ? "normal" : "preview",
  });
  const name = Form.useFieldValue(path.name, { ctx: form });
  const plot = Range.useAddToNewPlot();
  const openCSV = Framer.useDownloadCSVModal();
  return (
    <Form.Form<typeof Ranger.formSchema> {...form}>
      <Flex.Box y className={CSS.BE("range-overview", "header")}>
        {range.parent != null && (
          <Flex.Box x align="center" className={CSS.BE("range-overview", "crumbs")}>
            <Crumbs rangeKey={range.parent} />
          </Flex.Box>
        )}
        <Flex.Box x align="center" justify="between" gap="large">
          <Form.TextField
            path={path.name}
            showLabel={false}
            className={CSS.BE("range-overview", "name")}
            inputProps={{
              variant: "text",
              level: "h1",
              placeholder: "Name",
              onlyChangeOnBlur: true,
              resetOnBlurIfEmpty: true,
            }}
            padHelpText={false}
          />
          <Flex.Box
            x
            align="center"
            gap="small"
            className={CSS.BE("range-overview", "actions")}
          >
            {canPlot && (
              <>
                <Button.Button
                  variant="outlined"
                  gap="small"
                  onClick={() => plot([rangeKey])}
                >
                  <Icon.LinePlot />
                  Plot
                </Button.Button>
                <Divider.Divider y />
              </>
            )}
            <Button.Copy
              text={() => `rng = client.ranges.retrieve(key="${rangeKey}")`}
              tooltip="Copy Python code"
              tooltipLocation="bottom"
              variant="text"
              textColor={9}
            >
              <Icon.Python />
            </Button.Copy>
            <Button.Copy
              text={() => `const rng = await client.ranges.retrieve("${rangeKey}");`}
              tooltip="Copy TypeScript code"
              tooltipLocation="bottom"
              variant="text"
              textColor={9}
            >
              <Icon.TypeScript />
            </Button.Copy>
            <Button.Button
              tooltip="Download as CSV"
              tooltipLocation="bottom"
              variant="text"
              textColor={9}
              onClick={() =>
                openCSV({
                  timeRange: form.get(path.timeRange).value,
                  name,
                  channels: [],
                  icon: <Icon.Range />,
                })
              }
            >
              <Icon.CSV />
            </Button.Button>
            <Core.CopyLinkToolbarButton
              name={name}
              resourceID={ranger.resourceID(rangeKey)}
              tooltip="Copy link"
              tooltipLocation="bottom"
              textColor={9}
            />
            <Range.FavoriteButton range={range} size="medium" />
          </Flex.Box>
        </Flex.Box>
        <Form.Field path={path.timeRange} padHelpText={false} showLabel={false}>
          {(p) => (
            <Ranger.Timeline
              level="p"
              variant="shadow"
              className={CSS.BE("range-overview", "timeline")}
              parent={parent?.timeRange.numeric}
              {...p}
            />
          )}
        </Form.Field>
        <Flex.Box x align="center" gap="small">
          <Swatch range={range} preview={!canEdit} />
          <Labels rangeKey={rangeKey} preview={!canEdit} />
        </Flex.Box>
      </Flex.Box>
    </Form.Form>
  );
};
