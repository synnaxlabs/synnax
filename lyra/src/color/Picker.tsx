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
  type ReactElement,
  type ReactNode,
  useEffect,
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
import { Divider } from "@/divider";
import { Flex } from "@/flex";
import { useDebouncedCallback } from "@/hooks";
import { Icon } from "@/icon";
import { Input } from "@/input";
import { Menu } from "@/menu";
import { keyboardMenuKey, openContextMenu } from "@/menu/keyboard";
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
  /**
   * Shows the view in place of the tabs, under a back button and the title. Back
   * returns to the tabs as they were, with the focus on the element that opened the
   * view. The view renders outside every palette, so it cannot call `usePalette`: give
   * it what it needs, such as a key.
   */
  openView: (title: string, view: ReactNode) => void;
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
  library: { palettes, named, onNamedChange, paletteEnd, swatchMenu, footer },
  className,
  ...rest
}: TabbedProps): ReactElement => {
  const handleChange = useChangeHandler(onChange);
  const [view, setView] = useState<View | null>(null);
  const root = useRef<HTMLDivElement>(null);
  const opener = useRef<HTMLElement | null>(null);
  useEffect(() => {
    if (view != null || opener.current == null) return;
    if (opener.current.isConnected) opener.current.focus();
    else root.current?.focus();
    opener.current = null;
  }, [view]);
  const frequent = useFrequent();
  const linked = named == null ? undefined : findNamed(palettes, named);
  const [tab, setTab] = useState<Tab>(
    linked != null || value == null ? "library" : "custom",
  );
  const [search, setSearch] = useState("");
  const autoSelected = value == null && linked == null;
  const ctx: PickerContextValue = {
    value: color.construct(value ?? fallback ?? color.TRANSPARENT),
    openView: (title, content) => {
      opener.current =
        document.activeElement instanceof HTMLElement ? document.activeElement : null;
      setView({ title, content });
    },
  };
  return (
    <PickerContext value={ctx}>
      <Flex.Box
        y
        gap={0}
        className={CSS.cls(
          CSS.B("color-picker"),
          CSS.BM("color-picker", "tabbed"),
          view != null && CSS.BM("color-picker", "viewing"),
          className,
        )}
        background={1}
        ref={root}
        tabIndex={-1}
        {...rest}
      >
        {view != null && (
          <>
            <Flex.Box
              x
              align="center"
              gap="small"
              className={CSS.cls(
                CSS.BE("color-picker", "header"),
                CSS.BE("color-picker", "view-header"),
              )}
            >
              <Button.Button
                variant="text"
                size="small"
                square
                textColor={9}
                onClick={() => setView(null)}
                autoFocus
                tooltip="Back"
                aria-label="Back"
              >
                <Icon.Arrow.Left />
              </Button.Button>
              <Text.Text level="small" weight={500} overflow="fade">
                {view.title}
              </Text.Text>
            </Flex.Box>
            <Flex.Box y gap="medium" className={CSS.BE("color-picker", "view")}>
              {view.content}
            </Flex.Box>
          </>
        )}
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
              paletteEnd={paletteEnd}
              swatchMenu={swatchMenu}
              footer={footer}
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
    </PickerContext>
  );
};

interface View {
  title: string;
  content: ReactNode;
}

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

interface PaletteSectionProps extends Pick<Library, "paletteEnd" | "swatchMenu"> {
  palette: Palette;
  /** The colors of the palette that match the search. */
  colors: Named[];
  linked?: string;
  onPick: (key: string) => void;
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
  paletteEnd,
  swatchMenu,
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
  const menuProps =
    swatchMenu == null
      ? {}
      : {
          onContextMenu: menu.open,
          onKeyDown: (e: KeyboardEvent<HTMLElement>) => {
            const key = keyboardMenuKey(e);
            if (key == null) return;
            e.preventDefault();
            if (key === "F10" && !e.repeat) openContextMenu(e.currentTarget);
          },
          onKeyUp: (e: KeyboardEvent<HTMLElement>) => {
            if (keyboardMenuKey(e) !== "ContextMenu") return;
            e.preventDefault();
            openContextMenu(e.currentTarget);
          },
        };
  return (
    <PaletteContext value={{ palette }}>
      <Menu.ContextMenu
        {...menu}
        menu={({ keys }) => (
          <PaletteContext
            value={{ palette, swatch: palette.colors.find((c) => c.key === keys[0]) }}
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
              onPointerEnter={() => setHovered(c.key)}
              onFocus={() => setHovered(c.key)}
              aria-label={c.name}
              {...menuProps}
            />
          ))}
          {paletteEnd}
        </Section>
      </Menu.ContextMenu>
    </PaletteContext>
  );
};

interface PalettesProps extends Pick<Library, "paletteEnd" | "swatchMenu" | "footer"> {
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
  paletteEnd,
  swatchMenu,
  footer,
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
          paletteEnd={paletteEnd}
          swatchMenu={swatchMenu}
        />
      ))}
      {footer}
    </Flex.Box>
  );
};
