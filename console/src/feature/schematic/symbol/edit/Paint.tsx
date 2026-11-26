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
import { Color as Base } from "@synnaxlabs/lyra/color";
import { Dialog } from "@synnaxlabs/lyra/dialog";
import { Flex } from "@synnaxlabs/lyra/flex";
import { Form } from "@synnaxlabs/lyra/form";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Select } from "@synnaxlabs/lyra/select";
import { Text } from "@synnaxlabs/lyra/text";
import { Theming } from "@synnaxlabs/lyra/theming";
import { Color, Schematic } from "@synnaxlabs/pluto";
import { color } from "@synnaxlabs/x";
import type { CSSProperties, ReactElement } from "react";

import { FLAGS } from "@/flags";
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
  /** inherited is the paint that a look of the whole drawing, or of a state, gives. */
  inherited?: Value;
  /** inheritedFrom names the other state whose look gives inherited, if one does. */
  inheritedFrom?: string;
  /** own is the paint of the part itself, which Drawing clears. */
  own?: Value;
  /** lock says why the paint cannot change, if it cannot. */
  lock?: string;
  "aria-label": string;
}

const NAMES: Record<Kind, string> = {
  drawing: "Drawing",
  color: "Color",
  entry: "State color",
  contrast: "Contrast",
  node: "Node color",
};

const KINDS: Kind[] = ["drawing", "color", "entry", "contrast", "node"];

const NOTES: Record<Exclude<Kind, "color">, string> = {
  drawing: "The shapes keep the colors of the drawing.",
  entry: "Each state draws the color of its enumeration entry.",
  contrast: "Each state draws black or white, whichever reads on its entry color.",
  node: "Each node on a schematic sets this color.",
};

// FILLS are the button inputs for its fill at rest, on hover, and on press.
const FILLS = ["--pluto-bg", "--pluto-hover-bg", "--pluto-active-bg"];

const ICONS: Partial<Record<Kind, ReactElement>> = {
  entry: <Icon.Enumeration />,
  contrast: <Icon.Contrast />,
  node: <Icon.Node />,
};

/**
 * Paint picks the paint of a fill or stroke: the color of the drawing, a color, the
 * color of the entry of a state, black or white on that color, or the color of the
 * node.
 */
export const Paint = ({
  value,
  onChange,
  entryState,
  drawingColor,
  entryColor,
  inherited,
  inheritedFrom,
  own,
  lock,
  "aria-label": ariaLabel,
}: PaintProps): ReactElement => {
  const { gray } = Theming.use().colors;
  const kind: Kind = value?.type ?? "drawing";
  const variant = Form.useFieldValue(Schematic.Symbol.formPath.data.variant);
  // Until symbol nodes ship, only an actuator node gives a node color.
  const nodes = FLAGS.symbolNodes || variant === "actuator";
  // Drawing clears the paint of the part, which then takes the paint of the whole
  // drawing when there is one.
  const locked = inherited != null && own == null;
  const whole = inherited != null && own != null;
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
    if (next === "entry" || next === "contrast") {
      if (entryState != null) onChange({ type: next, state: entryState });
      return;
    }
    const start = shown == null || color.opacity(shown) === 0 ? gray.l8 : shown;
    onChange({
      type: "color",
      value: { type: "fixed", value: color.construct(start) },
    });
  };
  const opacity = shown == null ? 0 : color.opacity(shown);
  // A clear paint keeps the fill and text of the button. Any other paint sets the rest,
  // hover, and press fills, and the icon takes the gray that reads over it.
  let chip: CSSProperties | undefined;
  let text: color.Crude | 9 = 9;
  if (shown != null && opacity > 0) {
    const fill = color.cssString(shown);
    chip = Object.fromEntries(FILLS.map((v) => [v, fill]));
    text = color.pickByContrast(color.over(shown, gray.l0, opacity), gray.l0, gray.l11);
  }
  return (
    <Dialog.Frame variant="floating">
      <Dialog.Trigger
        hideCaret
        square
        size="small"
        variant="outlined"
        aria-label={ariaLabel}
        tooltip={lock ?? `${ariaLabel}: ${NAMES[kind]}`}
        disabled={lock != null}
        style={chip}
        textColor={text}
      >
        {ICONS[kind] ?? (shown == null && <Icon.Auto />)}
      </Dialog.Trigger>
      <Dialog.Dialog>
        <Flex.Box y gap="small" className={CSS.B("schematic-paint-dialog")}>
          <Select.Buttons<Kind> value={kind} onChange={handleKindChange} size="small">
            {KINDS.filter((k) => k !== "node" || nodes || kind === "node").map((k) => (
              <Select.Item
                key={k}
                itemKey={k}
                size="small"
                disabled={
                  (k === "drawing" && locked) ||
                  ((k === "entry" || k === "contrast") && entryState == null)
                }
                tooltip={
                  k === "drawing" && locked
                    ? `${inheritedFrom ?? "The whole drawing"} gives this paint. ` +
                      "Change it there."
                    : undefined
                }
              >
                {k === "drawing" && whole
                  ? (inheritedFrom ?? "Whole drawing")
                  : NAMES[k]}
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
