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
import {
  type ClipboardEvent,
  type ComponentPropsWithoutRef,
  type KeyboardEvent,
  type ReactElement,
  useCallback,
  useMemo,
  useState,
} from "react";

import { Button } from "@/button";
import { BaseSwatch } from "@/color/BaseSwatch";
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
   * optional and adds an Auto swatch that clears it.
   */
  fallback?: color.Crude;
}

const HUE_TRACK =
  "linear-gradient(to right, #f00 0%, #ff0 17%, #0f0 33%, #0ff 50%, #00f 67%, #f0f 83%, #f00 100%)";

/**
 * An HSV color picker: theme and palette swatches, a saturation and brightness plane,
 * hue and alpha sliders, a hex and alpha input row, and the user's recent colors.
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

  const handleChange = useCallback(
    (next: color.Color) => {
      onChange(next);
      updateFrequentDebounced(next);
    },
    [onChange, updateFrequentDebounced],
  );

  const handleHSVAChange = useCallback(
    (next: color.HSVA) => {
      hsva.set(next);
      handleChange(color.fromHSVA(next));
    },
    [hsva.set, handleChange],
  );

  // Moving the plane or the hue of a fully transparent color would change nothing
  // visible, so those moves make it opaque.
  const visibleAlpha = a === 0 ? 1 : a;
  const opaque = color.fromHSVA([h, s, v, 1]);
  const opaqueCSS = color.rgbaCSS(opaque);

  return (
    <Flex.Box
      y
      gap="medium"
      className={CSS.cls(CSS.B("color-picker"), className)}
      background={1}
      {...rest}
    >
      <Swatches
        value={value}
        fallback={fallback}
        onChange={onChange}
        onPick={handleChange}
      />
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
      <Recent value={value} onPick={handleChange} />
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
  onPick: (value: color.Color) => void;
}

const Swatches = ({
  value,
  fallback,
  onChange,
  onPick,
}: SwatchesProps): ReactElement => {
  const theme = Theming.use();
  const presets = useMemo(
    () => [
      theme.colors.primary.z,
      theme.colors.secondary.z,
      theme.colors.warning.z,
      theme.colors.error.z,
      ...theme.colors.visualization.palettes.default,
    ],
    [theme],
  );
  return (
    <div className={CSS.BE("color-picker", "swatches")}>
      {fallback != null && (
        <BaseSwatch
          value={fallback}
          className={CSS.cls(
            CSS.BE("color-picker", "swatch"),
            CSS.selected(value == null),
          )}
          draggable={false}
          onClick={() => onChange(undefined)}
          tooltip="Auto: the theme picks the color"
          aria-label="Auto"
        >
          <Icon.Auto />
        </BaseSwatch>
      )}
      {presets.map((c) => (
        <PickerSwatch key={color.hex(c)} value={c} current={value} onPick={onPick} />
      ))}
    </div>
  );
};

const Recent = ({
  value,
  onPick,
}: Pick<SwatchesProps, "value" | "onPick">): ReactElement | null => {
  const frequent = useFrequent();
  if (frequent.length === 0) return null;
  return (
    <>
      <Divider.Divider x />
      <div className={CSS.BE("color-picker", "swatches")}>
        {frequent.map((c) => (
          <PickerSwatch key={color.hex(c)} value={c} current={value} onPick={onPick} />
        ))}
      </div>
    </>
  );
};

interface PickerSwatchProps {
  value: color.Crude;
  current?: color.Crude;
  onPick: (value: color.Color) => void;
}

const PickerSwatch = ({ value, current, onPick }: PickerSwatchProps): ReactElement => (
  <BaseSwatch
    value={value}
    className={CSS.cls(
      CSS.BE("color-picker", "swatch"),
      CSS.selected(current != null && color.equals(current, value)),
    )}
    onClick={() => onPick(color.construct(value))}
    aria-label={color.hex(value)}
  />
);

interface InputRowProps {
  value: color.Crude;
  onChange: (value: color.Color) => void;
}

const HEX_DIGITS = /^#?[0-9a-f]+$/i;
const HEX_WITHOUT_ALPHA = /^#?([0-9a-f]{3}|[0-9a-f]{6})$/i;
const COMPLETE_HEX = /^#?([0-9a-f]{6}|[0-9a-f]{8})$/i;

/**
 * Parses what the user typed or pasted into the hex box. Bare hex digits need no `#`,
 * and any other CSS color also parses. A hex with no alpha digits keeps `alpha`.
 */
const parseHexInput = (text: string, alpha: number): color.Color | undefined => {
  const trimmed = text.trim();
  const isHex = HEX_DIGITS.test(trimmed);
  const parsed = color.fromCSS(
    isHex && !trimmed.startsWith("#") ? `#${trimmed}` : trimmed,
  );
  if (parsed == null) return undefined;
  return HEX_WITHOUT_ALPHA.test(trimmed) ? color.setAlpha(parsed, alpha) : parsed;
};

const InputRow = ({ value, onChange }: InputRowProps): ReactElement => {
  const alpha = color.aValue(value);
  const hex = color.hex(color.setAlpha(value, 1)).slice(1);
  const [draft, setDraft] = useState<string | null>(null);

  const commit = (text: string): void => {
    setDraft(null);
    const parsed = parseHexInput(text, alpha);
    if (parsed != null) onChange(parsed);
  };

  const handleDraftChange = (text: string): void => {
    setDraft(text);
    // A complete hex applies as the user types. Shorter forms wait for Enter or blur,
    // since "fff" is also the start of "ffffff".
    if (!COMPLETE_HEX.test(text.trim())) return;
    const parsed = parseHexInput(text, alpha);
    if (parsed != null) onChange(parsed);
  };

  const handlePaste = (e: ClipboardEvent<HTMLInputElement>): void => {
    const parsed = parseHexInput(e.clipboardData.getData("text"), alpha);
    if (parsed == null) return;
    e.preventDefault();
    setDraft(null);
    onChange(parsed);
  };

  const handleKeyDown = (e: KeyboardEvent<HTMLInputElement>): void => {
    if (e.key === "Enter" && draft != null) commit(draft);
  };

  return (
    <Flex.Box x gap="small" align="center">
      <Input.Text
        aria-label="Hex"
        size="small"
        grow
        value={draft ?? hex}
        onChange={handleDraftChange}
        onBlur={() => draft != null && commit(draft)}
        onKeyDown={handleKeyDown}
        onPaste={handlePaste}
        startContent="#"
        selectOnFocus
      />
      <Input.Numeric
        aria-label="Alpha percentage"
        className={CSS.BE("color-picker", "alpha-input")}
        size="small"
        value={Math.round(alpha * 100)}
        bounds={{ lower: 0, upper: 100 }}
        units="%"
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
