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
   * Links the value to the named color with the key, as a click on its swatch does.
   * The focus moves to its swatch, at once or when the swatch renders.
   */
  onNamedChange: (key: string) => void;
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
  /** The color that the palette title names: the hovered one, else the linked one. */
  shown?: Named;
}

const [PaletteContext, usePaletteContext] = context.create<PaletteContextValue>({
  displayName: "Color.PaletteContext",
  providerName: "paletteEnd or swatchActions",
});

/**
 * @returns the palette around a `paletteEnd` or `swatchActions` slot.
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
  library: { palettes, named, onNamedChange, paletteEnd, swatchActions, footer },
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
  const swatches = useRef(new Map<string, HTMLElement>());
  // The key of a linked swatch that takes the focus when it renders.
  const focusing = useRef<string>(undefined);
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
  const ctx: PickerContextValue = {
    value: color.construct(value ?? fallback ?? color.TRANSPARENT),
    linked,
    onNamedChange: (key) => {
      onNamedChange(key);
      const swatch = swatches.current.get(key);
      if (swatch == null) focusing.current = key;
      else swatch.focus();
    },
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
              viewing={view != null}
              linked={linked?.key}
              onPick={onNamedChange}
              registerSwatch={registerSwatch}
              paletteEnd={paletteEnd}
              swatchActions={swatchActions}
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
  /** Shown after the name. */
  actions?: ReactNode;
}

const Section = ({
  title,
  name,
  actions,
  children,
  ...rest
}: SectionProps): ReactElement => (
  <Flex.Box y gap="small" {...rest}>
    <Flex.Box
      x
      justify="between"
      align="center"
      gap="medium"
      className={CSS.BE("color-picker", "title")}
    >
      <Text.Text level="small" color={9} overflow="nowrap">
        {title}
      </Text.Text>
      <Flex.Box x align="center" gap="small" className={CSS.BE("color-picker", "name")}>
        <Text.Text level="small" color={10} overflow="fade">
          {name}
        </Text.Text>
        {actions}
      </Flex.Box>
    </Flex.Box>
    <div className={CSS.BE("color-picker", "grid")}>{children}</div>
  </Flex.Box>
);

interface PaletteSectionProps extends Pick<Library, "paletteEnd" | "swatchActions"> {
  palette: Palette;
  /** The colors of the palette that match the search. */
  colors: Named[];
  linked?: string;
  onPick: (key: string) => void;
  /** Records the swatch of a named color when it renders, and null when it goes. */
  registerSwatch: (key: string, swatch: HTMLElement | null) => void;
  /** If true, a view hides the palette, and the hovered color stays. */
  viewing: boolean;
}

/**
 * A palette under its title, which names the hovered color, else the linked one. The
 * hovered color stays until the pointer or the focus leaves the palette, so the actions
 * after its name stay in reach. It also stays under a view, so Back finds its actions.
 */
const PaletteSection = ({
  palette,
  colors,
  linked,
  onPick,
  registerSwatch,
  paletteEnd,
  swatchActions,
  viewing,
}: PaletteSectionProps): ReactElement => {
  const t = Theming.use();
  const [hovered, setHovered] = useState<string>();
  const leave = (): void => {
    if (!viewing) setHovered(undefined);
  };
  const shown =
    colors.find((c) => c.key === hovered) ?? colors.find((c) => c.key === linked);
  return (
    <PaletteContext value={{ palette, shown }}>
      <Section
        title={palette.name}
        name={shown?.name}
        actions={shown == null ? undefined : swatchActions}
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
            className={CSS.cls(CSS.selected(c.key === linked))}
            onClick={() => onPick(c.key)}
            ref={(el: HTMLElement | null) => registerSwatch(c.key, el)}
            onPointerEnter={() => setHovered(c.key)}
            onFocus={() => setHovered(c.key)}
            aria-label={c.name}
          />
        ))}
        {paletteEnd}
      </Section>
    </PaletteContext>
  );
};

interface PalettesProps
  extends
    Pick<Library, "paletteEnd" | "swatchActions" | "footer">,
    Pick<PaletteSectionProps, "registerSwatch"> {
  palettes: Palette[];
  search: string;
  /** The key of the named color that the value follows. */
  linked?: string;
  onPick: (key: string) => void;
  viewing: boolean;
}

/** The palettes, filtered by color name, palette name, or the hex a swatch shows. */
const Palettes = ({
  palettes,
  search,
  linked,
  onPick,
  registerSwatch,
  paletteEnd,
  swatchActions,
  footer,
  viewing,
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
          viewing={viewing}
          paletteEnd={paletteEnd}
          swatchActions={swatchActions}
        />
      ))}
      {footer}
    </Flex.Box>
  );
};
