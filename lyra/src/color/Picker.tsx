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
  type KeyboardEvent,
  type MouseEvent,
  type ReactElement,
  useRef,
  useState,
} from "react";

import { Button } from "@/button";
import { BaseSwatch } from "@/color/BaseSwatch";
import { Editor } from "@/color/Editor";
import { findNamed, type Library, type Named, type Palette } from "@/color/named";
import { useFrequent, useFrequentUpdater } from "@/color/Provider";
import { Swatches } from "@/color/Swatches";
import { context } from "@/context";
import { CSS } from "@/css";
import { Dialog } from "@/dialog";
import { Divider } from "@/divider";
import { Flex } from "@/flex";
import { useDebouncedCallback } from "@/hooks";
import { Icon } from "@/icon";
import { Input } from "@/input";
import { Menu } from "@/menu";
import { menuKeyDown, menuKeyUp, openContextMenu } from "@/menu/keyboard";
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

export interface PickerContextValue {
  /** The color that the picker shows: its value, else its fallback. */
  value: color.Color;
  /** The named color that the value links to. */
  linked?: Named;
  /**
   * Links the value to the named color with the key, as a click on its swatch does. The
   * Library tab shows with no search, and the focus moves to the swatch, at once or
   * when it renders.
   */
  link: (key: string) => void;
  /** Clears the search, so that every palette and color shows. */
  clearSearch: () => void;
  /**
   * Moves the focus into the `paletteEnd` slot of the palette with the key, once that
   * palette renders. The slot reads `focused` from `usePalette` and takes the focus.
   */
  focusPalette: (key: string) => void;
}

const [PickerContext, usePickerContext] = context.create<PickerContextValue>({
  displayName: "Color.PickerContext",
  providerName: "a library slot of Color.Picker",
});

/**
 * @returns the picker around a slot of its library.
 * @throws {Error} outside a slot of a library.
 */
export const usePicker = (): PickerContextValue => usePickerContext("Color.usePicker");

export interface PaletteContextValue {
  /** The palette, with every color it holds. */
  palette: Palette;
  /** In `swatchMenu`, the color of the swatch that the menu opened on. */
  swatch?: Named;
  /**
   * In `paletteEnd`, true if the slot takes the focus. It stays true until the focus
   * enters the palette.
   */
  focused: boolean;
}

const [PaletteContext, usePaletteContext] = context.create<PaletteContextValue>({
  displayName: "Color.PaletteContext",
  providerName: "paletteEnd or swatchMenu",
});

/**
 * @returns the palette around a `paletteEnd` or `swatchMenu` slot.
 * @throws {Error} outside those slots.
 */
export const usePalette = (): PaletteContextValue =>
  usePaletteContext("Color.usePalette");

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
  library: { palettes, named, onNamedChange, footer, customFooter, ...slots },
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
  const swatches = useRef(new Map<string, HTMLElement>());
  // The key of a linked swatch that takes the focus when it renders.
  const focusing = useRef<string>(undefined);
  const [focusedPalette, setFocusedPalette] = useState<string>();
  const registerSwatch = (key: string, swatch: HTMLElement | null): void => {
    if (swatch == null) {
      swatches.current.delete(key);
      return;
    }
    swatches.current.set(key, swatch);
    if (focusing.current !== key) return;
    focusing.current = undefined;
    swatch.focus();
  };
  const autoSelected = value == null && linked == null;
  const tabFooter = tab === "library" ? footer : customFooter;
  const ctx: PickerContextValue = {
    value: color.construct(value ?? fallback ?? color.TRANSPARENT),
    linked,
    link: (key) => {
      setTab("library");
      setSearch("");
      onNamedChange(key);
      const swatch = swatches.current.get(key);
      if (swatch == null) focusing.current = key;
      else swatch.focus();
    },
    clearSearch: () => setSearch(""),
    focusPalette: setFocusedPalette,
  };
  return (
    <PickerContext value={ctx}>
      <Dialog.Views
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
              registerSwatch={registerSwatch}
              focusedPalette={focusedPalette}
              onPaletteFocus={() => setFocusedPalette(undefined)}
              {...slots}
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
        {tabFooter != null && (
          <Flex.Box y empty className={CSS.BE("color-picker", "footer")}>
            {tabFooter}
          </Flex.Box>
        )}
      </Dialog.Views>
    </PickerContext>
  );
};

interface SectionProps extends Omit<Flex.BoxProps, "title"> {
  title: string;
  /** Shown at the end of the title. */
  name?: string;
}

const Section = ({ title, name, children, ...rest }: SectionProps): ReactElement => (
  <Flex.Box y gap="small" {...rest}>
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

interface PaletteSectionProps extends Pick<
  Library,
  "paletteEnd" | "swatchMenu" | "hasSwatchMenu"
> {
  palette: Palette;
  /** The colors of the palette that match the search. */
  colors: Named[];
  linked?: string;
  onPick: (key: string) => void;
  /** Records the swatch of a named color when it renders, and null when it goes. */
  registerSwatch: (key: string, swatch: HTMLElement | null) => void;
  /** If true, the `paletteEnd` slot takes the focus. */
  focused: boolean;
  /** Called when the focus enters the palette while it is focused. */
  onFocus: () => void;
}

/**
 * A palette under its title, which names the color of an open swatch menu, else the
 * hovered or focused color, else the linked one. A hovered name stays until the pointer
 * or the focus leaves the palette.
 */
const PaletteSection = ({
  palette,
  colors,
  linked,
  onPick,
  registerSwatch,
  focused,
  onFocus,
  paletteEnd,
  swatchMenu,
  hasSwatchMenu,
}: PaletteSectionProps): ReactElement => {
  const t = Theming.use();
  const [hovered, setHovered] = useState<string>();
  const menu = Menu.useContextMenu();
  const leave = (): void => setHovered(undefined);
  const find = (key?: string): Named | undefined =>
    key == null ? undefined : colors.find((c) => c.key === key);
  // The menu takes the focus, so the title names its swatch while it is open.
  const shown =
    find(menu.visible ? menu.keys[0] : undefined) ?? find(hovered) ?? find(linked);
  const menuPropsOf = (key: string) => {
    if (swatchMenu == null) return {};
    const opens = (): boolean => hasSwatchMenu?.(key) ?? true;
    return {
      onContextMenu: (e: MouseEvent<HTMLElement>) => {
        if (opens()) menu.open(e);
      },
      onKeyDown: (e: KeyboardEvent<HTMLElement>) => {
        if (opens()) menuKeyDown(e, () => openContextMenu(e.currentTarget));
      },
      onKeyUp: (e: KeyboardEvent<HTMLElement>) => {
        if (opens()) menuKeyUp(e, () => openContextMenu(e.currentTarget));
      },
    };
  };
  return (
    <PaletteContext value={{ palette, focused }}>
      <Menu.ContextMenu
        {...menu}
        menu={({ keys }) => (
          <PaletteContext
            value={{
              palette,
              swatch: palette.colors.find((c) => c.key === keys[0]),
              focused: false,
            }}
          >
            {swatchMenu}
          </PaletteContext>
        )}
      >
        <Section
          title={palette.name}
          name={shown?.name}
          role="group"
          aria-label={palette.name}
          onPointerLeave={leave}
          onFocus={() => {
            if (focused) onFocus();
          }}
          onBlur={(e) => {
            if (!e.currentTarget.contains(e.relatedTarget)) leave();
          }}
        >
          {colors.map((c) => (
            <BaseSwatch
              key={c.key}
              value={theme.resolve(c, t)}
              size="tiny"
              draggable={false}
              className={CSS.cls(
                CSS.selected(c.key === linked),
                swatchMenu != null && Menu.CONTEXT_TARGET,
              )}
              data-menu-key={c.key}
              onClick={() => onPick(c.key)}
              ref={(el: HTMLElement | null) => registerSwatch(c.key, el)}
              onPointerEnter={() => setHovered(c.key)}
              onFocus={() => setHovered(c.key)}
              aria-label={c.name}
              {...menuPropsOf(c.key)}
            />
          ))}
          {paletteEnd}
        </Section>
      </Menu.ContextMenu>
    </PaletteContext>
  );
};

interface PalettesProps
  extends
    Pick<Library, "paletteEnd" | "swatchMenu" | "hasSwatchMenu">,
    Pick<PaletteSectionProps, "registerSwatch"> {
  palettes: Palette[];
  search: string;
  /** The key of the named color that the value follows. */
  linked?: string;
  onPick: (key: string) => void;
  /** The key of the palette whose `paletteEnd` slot takes the focus. */
  focusedPalette?: string;
  /** Called when the focus enters the focused palette. */
  onPaletteFocus: () => void;
}

/** The palettes, filtered by color name, palette name, or the hex a swatch shows. */
const Palettes = ({
  palettes,
  search,
  linked,
  onPick,
  registerSwatch,
  focusedPalette,
  onPaletteFocus,
  paletteEnd,
  swatchMenu,
  hasSwatchMenu,
}: PalettesProps): ReactElement => {
  const t = Theming.use();
  const term = search.trim().toLowerCase();
  const shown = palettes
    .map((palette) => ({
      palette,
      colors:
        term.length === 0 || palette.name.toLowerCase().includes(term)
          ? palette.colors
          : palette.colors.filter(
              (c) =>
                c.name.toLowerCase().includes(term) ||
                color.hex(theme.resolve(c, t)).includes(term.replace("#", "")),
            ),
    }))
    .filter(
      ({ colors }) => colors.length > 0 || (term.length === 0 && paletteEnd != null),
    );
  return (
    <Flex.Box y gap={2} className={CSS.BE("color-picker", "palettes")}>
      {shown.length === 0 && (
        <Status.Summary center variant="disabled" hideIcon>
          {term.length === 0 ? "No colors found" : `No colors match "${search.trim()}"`}
        </Status.Summary>
      )}
      {shown.map(({ palette, colors }) => (
        <PaletteSection
          key={palette.key}
          palette={palette}
          colors={colors}
          linked={linked}
          onPick={onPick}
          registerSwatch={registerSwatch}
          focused={palette.key === focusedPalette}
          onFocus={onPaletteFocus}
          paletteEnd={paletteEnd}
          swatchMenu={swatchMenu}
          hasSwatchMenu={hasSwatchMenu}
        />
      ))}
    </Flex.Box>
  );
};
