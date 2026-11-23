// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/feature/schematic/symbol/edit/Edit.css";

import { type schematic } from "@synnaxlabs/client";
import { Color as Base } from "@synnaxlabs/lyra/color";
import { Dialog } from "@synnaxlabs/lyra/dialog";
import { Flex } from "@synnaxlabs/lyra/flex";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Select } from "@synnaxlabs/lyra/select";
import { Text } from "@synnaxlabs/lyra/text";
import { Theming } from "@synnaxlabs/lyra/theming";
import { Color } from "@synnaxlabs/pluto";
import { color } from "@synnaxlabs/x";
import { type ReactElement } from "react";

import { CSS } from "@/platform/css";

type Value = schematic.symbol.Paint;
type Kind = "drawing" | Value["type"];

export interface PaintProps {
  value?: Value;
  onChange: (value?: Value) => void;
  /** entryState is the state whose entries color `entry` paint, if it has any. */
  entryState?: string;
  /** drawingColor is the color that the drawing gives the shapes. */
  drawingColor?: color.Crude;
  /** entryColor is the color of the entry that the editor shows. */
  entryColor?: color.Crude;
  "aria-label": string;
}

const NAMES: Record<Kind, string> = {
  drawing: "Drawing",
  color: "Color",
  entry: "State color",
  node: "Node color",
};

const NOTES: Record<Exclude<Kind, "color">, string> = {
  drawing: "The shapes keep the colors of the drawing.",
  entry: "Each state draws the color of its enumeration entry.",
  node: "Each node on a schematic sets this color.",
};

const ICONS: Partial<Record<Kind, ReactElement>> = {
  entry: <Icon.Enumeration />,
  node: <Icon.Node />,
};

/**
 * Paint picks the paint of a fill or stroke: the color of the drawing, a color, the
 * color of the entry of a state, or the color of the node.
 */
export const Paint = ({
  value,
  onChange,
  entryState,
  drawingColor,
  entryColor,
  "aria-label": ariaLabel,
}: PaintProps): ReactElement => {
  const { gray } = Theming.use().colors;
  const kind: Kind = value?.type ?? "drawing";
  const binding = Color.useBinding(
    value?.type === "color" ? value.value : undefined,
    (ref) => onChange(ref == null ? undefined : { type: "color", value: ref }),
  );
  let shown: color.Crude | undefined;
  if (kind === "drawing") shown = drawingColor;
  else if (kind === "color") shown = binding.value;
  else if (kind === "entry") shown = entryColor;
  const handleKindChange = (next: Kind) => {
    if (next === kind) return;
    if (next === "drawing") return onChange(undefined);
    if (next === "node") return onChange({ type: "node" });
    if (next === "entry") {
      if (entryState != null) onChange({ type: "entry", state: entryState });
      return;
    }
    const start = shown == null || color.opacity(shown) === 0 ? gray.l8 : shown;
    onChange({
      type: "color",
      value: { type: "fixed", value: color.construct(start) },
    });
  };
  const background = shown ?? color.TRANSPARENT;
  return (
    <Dialog.Frame variant="floating">
      <Dialog.Trigger
        hideCaret
        square
        size="small"
        variant="outlined"
        aria-label={ariaLabel}
        tooltip={NAMES[kind]}
        style={{
          backgroundColor: color.cssString(background),
          color: color.cssString(color.pickByContrast(background, gray.l0, gray.l11)),
        }}
      >
        {ICONS[kind] ?? (shown == null && <Icon.Auto />)}
      </Dialog.Trigger>
      <Dialog.Dialog>
        <Flex.Box y gap="small" className={CSS.B("schematic-paint-dialog")}>
          <Select.Buttons<Kind> value={kind} onChange={handleKindChange} size="small">
            {(Object.keys(NAMES) as Kind[]).map((k) => (
              <Select.Item
                key={k}
                itemKey={k}
                size="small"
                disabled={k === "entry" && entryState == null}
              >
                {NAMES[k]}
              </Select.Item>
            ))}
          </Select.Buttons>
          {kind === "color" ? (
            <Base.Picker
              value={binding.value}
              onChange={binding.onChange}
              library={binding.library}
            />
          ) : (
            <Text.Text level="small" color={9}>
              {NOTES[kind]}
            </Text.Text>
          )}
        </Flex.Box>
      </Dialog.Dialog>
    </Dialog.Frame>
  );
};
