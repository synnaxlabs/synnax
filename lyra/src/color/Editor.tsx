// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { color, errors } from "@synnaxlabs/x";
import { type ReactElement, useState } from "react";

import { Button } from "@/button";
import { HexText, parseHexInput } from "@/color/HexText";
import { Plane } from "@/color/Plane";
import { Slider } from "@/color/Slider";
import { CSS } from "@/css";
import { Flex } from "@/flex";
import { Icon } from "@/icon";
import { Input } from "@/input";
import { Status } from "@/status";

const HUE_STOPS = ["#f00", "#ff0", "#0f0", "#0ff", "#00f", "#f0f", "#f00"];
const HUE_TRACK = `linear-gradient(to right, ${HUE_STOPS.join(", ")})`;

export interface EditorProps {
  value: color.Crude;
  onChange: (value: color.Color) => void;
}

/** A saturation and brightness plane, hue and alpha sliders, and the hex input row. */
export const Editor = ({ value, onChange }: EditorProps): ReactElement => {
  const hsva = useHSVA(value);
  const [h, s, v, a] = hsva.value;
  const handleHSVAChange = (next: color.HSVA): void => {
    hsva.set(next);
    onChange(color.fromHSVA(next));
  };
  // Moving the plane or the hue of a fully transparent color would change nothing
  // visible, so those moves make it opaque.
  const visibleAlpha = a === 0 ? 1 : a;
  const opaqueCSS = color.cssString(color.fromHSVA([h, s, v, 1]));
  return (
    <>
      <Flex.Box y gap="small">
        <Plane
          value={hsva.value}
          thumb={opaqueCSS}
          onChange={(nextS, nextV) => handleHSVAChange([h, nextS, nextV, visibleAlpha])}
        />
        <Slider
          label="Hue"
          value={h / 360}
          valueText={`${Math.round(h)} degrees`}
          track={HUE_TRACK}
          thumb={color.cssString(color.fromHSVA([h, 100, 100, 1]))}
          onChange={(nextH) => handleHSVAChange([nextH * 360, s, v, visibleAlpha])}
        />
        <Slider
          label="Alpha"
          className={CSS.BE("color-picker", "alpha")}
          value={a}
          valueText={`${Math.round(a * 100)}%`}
          track={`linear-gradient(to right, transparent, ${opaqueCSS})`}
          thumb={color.cssString(color.fromHSVA(hsva.value))}
          onChange={(nextA) => handleHSVAChange([h, s, v, nextA])}
        />
      </Flex.Box>
      <InputRow value={value} onChange={onChange} />
    </>
  );
};

interface UseHSVAReturn {
  value: color.HSVA;
  set: (value: color.HSVA) => void;
}

/**
 * Holds the color as HSVA so the hue survives a move to gray, black, or white, where
 * RGB cannot carry it. Resyncs when the value changes to a color the HSVA does not
 * produce.
 */
const useHSVA = (value: color.Crude): UseHSVAReturn => {
  const [hsva, setHSVA] = useState(() => color.hsva(value));
  const [prev, setPrev] = useState(value);
  if (!color.equals(prev, value)) {
    setPrev(value);
    if (!color.equals(color.fromHSVA(hsva), value)) setHSVA(color.hsva(value));
  }
  return { value: hsva, set: setHSVA };
};

interface InputRowProps {
  value: color.Crude;
  onChange: (value: color.Color) => void;
}

const InputRow = ({ value, onChange }: InputRowProps): ReactElement => {
  const alpha = color.opacity(value);
  return (
    <Flex.Box x gap="tiny" align="center">
      <Flex.Box pack grow>
        <HexText
          aria-label="Hex"
          size="small"
          grow
          startContent="#"
          text={color.hex(color.setOpacity(value, 1)).slice(1)}
          alpha={alpha}
          onChange={onChange}
        />
        <Input.Numeric
          aria-label="Alpha percentage"
          className={CSS.BE("color-picker", "alpha-input")}
          size="small"
          value={Math.round(alpha * 100)}
          bounds={{ lower: 0, upper: 100 }}
          endContent="%"
          startContent={<Icon.Opacity />}
          showDragHandle={false}
          onChange={(pct) => onChange(color.setOpacity(value, pct / 100))}
        />
      </Flex.Box>
      <Button.Copy
        size="small"
        variant="text"
        text={() => color.hex(value)}
        tooltip="Copy hex"
      />
      <Eyedropper alpha={alpha} onChange={onChange} />
    </Flex.Box>
  );
};

interface EyeDropperResult {
  sRGBHex: string;
}

interface EyeDropper {
  open: () => Promise<EyeDropperResult>;
}

type EyeDropperConstructor = new () => EyeDropper;

const getEyeDropper = (): EyeDropperConstructor | undefined =>
  (globalThis as { EyeDropper?: EyeDropperConstructor }).EyeDropper;

/** @returns the picked color as hex, or null when the user leaves with Escape. */
const pickFromScreen = async (Ctor: EyeDropperConstructor): Promise<string | null> => {
  try {
    return (await new Ctor().open()).sRGBHex;
  } catch (e) {
    if (e instanceof DOMException && e.name === "AbortError") return null;
    throw errors.fromUnknown(e);
  }
};

interface EyedropperProps {
  alpha: number;
  onChange: (value: color.Color) => void;
}

/**
 * Picks a color from anywhere on the screen. Renders nothing where the platform has
 * no EyeDropper API, such as WebKit.
 */
const Eyedropper = ({ alpha, onChange }: EyedropperProps): ReactElement | null => {
  const handleError = Status.useErrorHandler();
  const Ctor = getEyeDropper();
  if (Ctor == null) return null;
  const handleClick = (): void =>
    handleError(async () => {
      const hex = await pickFromScreen(Ctor);
      if (hex == null) return;
      const parsed = parseHexInput(hex, alpha);
      if (parsed != null) onChange(parsed);
    }, "Failed to pick a color from the screen");
  return (
    <Button.Button
      size="small"
      variant="text"
      onClick={handleClick}
      tooltip="Pick a color from the screen"
      aria-label="Pick a color from the screen"
    >
      <Icon.Eyedropper />
    </Button.Button>
  );
};
