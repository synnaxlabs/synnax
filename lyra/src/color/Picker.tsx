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
  type ComponentPropsWithoutRef,
  type ReactElement,
  type ReactNode,
  useState,
} from "react";

import { Button } from "@/button";
import { BaseSwatch } from "@/color/BaseSwatch";
import { Editor } from "@/color/Editor";
import { findNamed, type Library, type Palette } from "@/color/named";
import { useFrequent, useFrequentUpdater } from "@/color/Provider";
import { Swatches } from "@/color/Swatches";
import { CSS } from "@/css";
import { Divider } from "@/divider";
import { Flex } from "@/flex";
import { useDebouncedCallback } from "@/hooks";
import { Icon } from "@/icon";
import { Input } from "@/input";
import { Select } from "@/select";
import { Status } from "@/status";
import { Text } from "@/text";
import { theme } from "@/theme";
import { Theming } from "@/theming";

export interface PickerProps
  extends
    Omit<ComponentPropsWithoutRef<"div">, "onChange" | "defaultValue">,
    Input.Control<color.Crude | undefined, color.Color | undefined> {
  /**
   * The color the theme paints while the value is absent. Setting it makes the value
   * optional and adds an Auto button that clears it.
   */
  fallback?: color.Crude;
  /**
   * Named colors to offer. Setting it splits the picker into two tabs: Library links
   * the value to a named color, and Custom edits a fixed color.
   */
  library?: Library;
}

/**
 * An HSV color picker: a saturation and brightness plane, hue and alpha sliders, a hex
 * and alpha input row, and swatches. With a library, the swatches are its named colors.
 */
export const Picker = ({ library, ...rest }: PickerProps): ReactElement =>
  library == null ? <Plain {...rest} /> : <Tabbed library={library} {...rest} />;

type BaseProps = Omit<PickerProps, "library">;

/** Wraps onChange so that each picked color also joins the recent colors. */
const useChangeHandler = (
  onChange: (c?: color.Color) => void,
): ((c?: color.Color) => void) => {
  const updateFrequent = useFrequentUpdater();
  const updateFrequentDebounced = useDebouncedCallback(
    updateFrequent,
    TimeSpan.SECOND,
    [updateFrequent],
  );
  return (next) => {
    onChange(next);
    if (next != null) updateFrequentDebounced(next);
  };
};

const Plain = ({
  value,
  onChange,
  fallback,
  className,
  ...rest
}: BaseProps): ReactElement => {
  const handleChange = useChangeHandler(onChange);
  return (
    <Flex.Box
      y
      gap="medium"
      className={CSS.cls(CSS.B("color-picker"), className)}
      background={1}
      {...rest}
    >
      <Editor value={value ?? fallback ?? color.TRANSPARENT} onChange={handleChange} />
      <Divider.Divider x />
      <Swatches value={value} fallback={fallback} onChange={handleChange} />
    </Flex.Box>
  );
};

type Tab = "library" | "custom";

interface TabbedProps extends BaseProps {
  library: Library;
}

const Tabbed = ({
  value,
  onChange,
  fallback,
  library: { palettes, named, onNamedChange },
  className,
  ...rest
}: TabbedProps): ReactElement => {
  const handleChange = useChangeHandler(onChange);
  const frequent = useFrequent();
  const linked = named == null ? undefined : findNamed(palettes, named);
  const [tab, setTab] = useState<Tab>(
    linked != null || value == null ? "library" : "custom",
  );
  const [search, setSearch] = useState("");
  const autoSelected = value == null && linked == null;
  return (
    <Flex.Box
      y
      gap={0}
      className={CSS.cls(
        CSS.B("color-picker"),
        CSS.BM("color-picker", "tabbed"),
        className,
      )}
      background={1}
      {...rest}
    >
      <Flex.Box
        x
        align="center"
        gap="small"
        className={CSS.BE("color-picker", "header")}
      >
        <Select.Buttons<Tab>
          value={tab}
          onChange={setTab}
          x
          className={CSS.BE("color-picker", "tabs")}
        >
          <Select.Item itemKey="library" size="small" level="small">
            <Icon.Library />
            Library
          </Select.Item>
          <Select.Item itemKey="custom" size="small" level="small">
            <Icon.Sliders />
            Custom
          </Select.Item>
        </Select.Buttons>
        {fallback != null && (
          <Button.Button
            variant="outlined"
            size="small"
            square
            className={CSS.cls(CSS.selected(autoSelected))}
            onClick={() => handleChange(undefined)}
            tooltip="Auto: the theme picks the color"
            aria-label="Auto"
          >
            <Icon.Auto />
          </Button.Button>
        )}
      </Flex.Box>
      {tab === "library" ? (
        <>
          <Input.Search
            value={search}
            onChange={setSearch}
            flush
            rounded
            full="x"
            size="medium"
            placeholder="Search colors"
            startContent={<Icon.Search />}
            className={CSS.BE("color-picker", "search")}
          />
          <Palettes
            palettes={palettes}
            search={search}
            linked={linked?.key}
            onPick={onNamedChange}
          />
        </>
      ) : (
        <Flex.Box y gap="medium" className={CSS.BE("color-picker", "custom")}>
          <Editor
            value={value ?? fallback ?? color.TRANSPARENT}
            onChange={handleChange}
          />
          {frequent.length > 0 && (
            <Section title="Recent">
              {frequent.map((c) => (
                <BaseSwatch
                  key={color.hex(c)}
                  value={c}
                  size="tiny"
                  draggable={false}
                  className={CSS.cls(
                    CSS.selected(
                      linked == null && value != null && color.equals(value, c),
                    ),
                  )}
                  onClick={() => handleChange(color.construct(c))}
                  aria-label={color.hex(c)}
                />
              ))}
            </Section>
          )}
        </Flex.Box>
      )}
    </Flex.Box>
  );
};

interface SectionProps {
  title: string;
  /** Shown at the end of the title. */
  name?: string;
  children: ReactNode;
}

const Section = ({ title, name, children }: SectionProps): ReactElement => (
  <Flex.Box y gap="small">
    <Flex.Box x justify="between" gap="medium">
      <Text.Text level="small" color={9} overflow="nowrap">
        {title}
      </Text.Text>
      <Text.Text level="small" color={10} overflow="fade">
        {name}
      </Text.Text>
    </Flex.Box>
    <div className={CSS.BE("color-picker", "grid")}>{children}</div>
  </Flex.Box>
);

interface PaletteSectionProps {
  palette: Palette;
  linked?: string;
  onPick: (key: string) => void;
}

/** A palette under its title, which names the hovered color, else the linked one. */
const PaletteSection = ({
  palette,
  linked,
  onPick,
}: PaletteSectionProps): ReactElement => {
  const t = Theming.use();
  const [hovered, setHovered] = useState<string>();
  const named =
    palette.colors.find((c) => c.key === hovered) ??
    palette.colors.find((c) => c.key === linked);
  return (
    <Section title={palette.name} name={named?.name}>
      {palette.colors.map((c) => (
        <BaseSwatch
          key={c.key}
          value={theme.resolve(c, t)}
          size="tiny"
          draggable={false}
          className={CSS.cls(CSS.selected(c.key === linked))}
          onClick={() => onPick(c.key)}
          onPointerEnter={() => setHovered(c.key)}
          onPointerLeave={() => setHovered((h) => (h === c.key ? undefined : h))}
          aria-label={c.name}
        />
      ))}
    </Section>
  );
};

interface PalettesProps {
  palettes: Palette[];
  search: string;
  /** The key of the named color that the value follows. */
  linked?: string;
  onPick: (key: string) => void;
}

/** The palettes, filtered by color name, palette name, or the hex a swatch shows. */
const Palettes = ({
  palettes,
  search,
  linked,
  onPick,
}: PalettesProps): ReactElement => {
  const t = Theming.use();
  const term = search.trim().toLowerCase();
  const shown = palettes
    .map((p) => ({
      ...p,
      colors:
        term.length === 0 || p.name.toLowerCase().includes(term)
          ? p.colors
          : p.colors.filter(
              (c) =>
                c.name.toLowerCase().includes(term) ||
                color.hex(theme.resolve(c, t)).includes(term.replace("#", "")),
            ),
    }))
    .filter((p) => p.colors.length > 0);
  return (
    <Flex.Box y gap={2} className={CSS.BE("color-picker", "palettes")}>
      {shown.length === 0 && (
        <Status.Summary center variant="disabled" hideIcon>
          {term.length === 0 ? "No colors found" : `No colors match "${search.trim()}"`}
        </Status.Summary>
      )}
      {shown.map((p) => (
        <PaletteSection key={p.key} palette={p} linked={linked} onPick={onPick} />
      ))}
    </Flex.Box>
  );
};
