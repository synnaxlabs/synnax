// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/color/Picker.css";

import { color, TimeSpan } from "@synnaxlabs/x";
import { type ComponentPropsWithoutRef, type ReactElement, useState } from "react";

import { Button } from "@/button";
import { BaseSwatch } from "@/color/BaseSwatch";
import { HexText, parseHexInput } from "@/color/HexText";
import { Plane } from "@/color/Plane";
import { useFrequent, useFrequentUpdater } from "@/color/Provider";
import { Slider } from "@/color/Slider";
import { CSS } from "@/css";
import { Divider } from "@/divider";
import { Flex } from "@/flex";
import { useDebouncedCallback } from "@/hooks";
import { Icon } from "@/icon";
import { Input } from "@/input";
import { Status } from "@/status";
import { Theming } from "@/theming";

export interface PickerProps extends Omit<
  ComponentPropsWithoutRef<"div">,
  "onChange" | "defaultValue"
> {
  /** The picked color. Absent only when {@link PickerProps.fallback} is set. */
  value?: color.Crude;
  /** Called with the new color, or with undefined when the user picks Auto. */
  onChange: (value?: color.Color) => void;
  /**
   * The color the theme paints while the value is absent. Setting it makes the value
   * optional and adds an Auto button that clears it.
   */
  fallback?: color.Crude;
}

const HUE_STOPS = ["#f00", "#ff0", "#0f0", "#0ff", "#00f", "#f0f", "#f00"];
const HUE_TRACK = `linear-gradient(to right, ${HUE_STOPS.join(", ")})`;

/**
 * An HSV color picker: a saturation and brightness plane, hue and alpha sliders, a hex
 * and alpha input row, theme and palette swatches, and the user's recent colors.
 */
export const Picker = ({
  value,
  onChange,
  fallback,
  className,
  ...rest
}: PickerProps): ReactElement => {
  const shown = value ?? fallback ?? color.ZERO;
  const hsva = useHSVA(shown);
  const [h, s, v, a] = hsva.value;
  const updateFrequent = useFrequentUpdater();
  const updateFrequentDebounced = useDebouncedCallback(
    updateFrequent,
    TimeSpan.SECOND,
    [updateFrequent],
  );

  const handleChange = (next?: color.Color): void => {
    onChange(next);
    if (next != null) updateFrequentDebounced(next);
  };

  const handleHSVAChange = (next: color.HSVA): void => {
    hsva.set(next);
    handleChange(color.fromHSVA(next));
  };

  // Moving the plane or the hue of a fully transparent color would change nothing
  // visible, so those moves make it opaque.
  const visibleAlpha = a === 0 ? 1 : a;
  const opaqueCSS = color.rgbaCSS(color.fromHSVA([h, s, v, 1]));

  return (
    <Flex.Box
      y
      gap="medium"
      className={CSS.cls(CSS.B("color-picker"), className)}
      background={1}
      {...rest}
    >
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
          thumb={color.rgbaCSS(color.fromHSVA([h, 100, 100, 1]))}
          onChange={(nextH) => handleHSVAChange([nextH * 360, s, v, visibleAlpha])}
        />
        <Slider
          label="Alpha"
          className={CSS.BE("color-picker", "alpha")}
          value={a}
          valueText={`${Math.round(a * 100)}%`}
          track={`linear-gradient(to right, transparent, ${opaqueCSS})`}
          thumb={color.rgbaCSS(color.fromHSVA(hsva.value))}
          onChange={(nextA) => handleHSVAChange([h, s, v, nextA])}
        />
      </Flex.Box>
      <InputRow value={shown} onChange={handleChange} />
      <Divider.Divider x />
      <Swatches value={value} fallback={fallback} onChange={handleChange} />
      <Recent value={value} onChange={handleChange} />
    </Flex.Box>
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

interface SwatchesProps {
  value?: color.Crude;
  fallback?: color.Crude;
  onChange: (value?: color.Color) => void;
}

const Swatches = ({ value, fallback, onChange }: SwatchesProps): ReactElement => {
  const { colors } = Theming.use();
  const presets = [
    colors.primary.z,
    colors.secondary.z,
    colors.warning.z,
    colors.error.z,
    ...colors.visualization.palettes.default,
  ];
  return (
    <div className={CSS.BE("color-picker", "swatches")}>
      {fallback != null && (
        <Button.Button
          variant="outlined"
          size="tiny"
          className={CSS.cls(
            CSS.BE("color-picker", "auto"),
            CSS.selected(value == null),
          )}
          onClick={() => onChange(undefined)}
          tooltip="Auto: the theme picks the color"
          aria-label="Auto"
        >
          <Icon.Auto />
          Auto
        </Button.Button>
      )}
      {presets.map((c) => (
        <PickerSwatch
          key={color.hex(c)}
          value={c}
          current={value}
          onChange={onChange}
        />
      ))}
    </div>
  );
};

const Recent = ({
  value,
  onChange,
}: Omit<SwatchesProps, "fallback">): ReactElement | null => {
  const frequent = useFrequent();
  if (frequent.length === 0) return null;
  return (
    <>
      <Divider.Divider x />
      <div className={CSS.BE("color-picker", "swatches")}>
        {frequent.map((c) => (
          <PickerSwatch
            key={color.hex(c)}
            value={c}
            current={value}
            onChange={onChange}
          />
        ))}
      </div>
    </>
  );
};

interface PickerSwatchProps {
  value: color.Crude;
  /** The picker's color. The swatch shows as selected when it matches. */
  current?: color.Crude;
  onChange: (value: color.Color) => void;
}

const PickerSwatch = ({
  value,
  current,
  onChange,
}: PickerSwatchProps): ReactElement => (
  <BaseSwatch
    value={value}
    className={CSS.cls(CSS.selected(current != null && color.equals(current, value)))}
    size="tiny"
    onClick={() => onChange(color.construct(value))}
    aria-label={color.hex(value)}
  />
);

interface InputRowProps {
  value: color.Crude;
  onChange: (value: color.Color) => void;
}

const InputRow = ({ value, onChange }: InputRowProps): ReactElement => {
  const alpha = color.aValue(value);
  return (
    <Flex.Box x gap="small" align="center">
      <HexText
        aria-label="Hex"
        size="small"
        grow
        startContent="#"
        text={color.hex(color.setAlpha(value, 1)).slice(1)}
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
        onChange={(pct) => onChange(color.setAlpha(value, pct / 100))}
      />
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
    throw e instanceof Error ? e : new Error(String(e), { cause: e });
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
