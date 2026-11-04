// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/platform/selector/Selector.css";

import { type panel } from "@synnaxlabs/client";
import { type CSS as PCSS } from "@synnaxlabs/lyra/css";
import { Flex } from "@synnaxlabs/lyra/flex";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Input } from "@synnaxlabs/lyra/input";
import { Portal } from "@synnaxlabs/lyra/portal";
import { Select } from "@synnaxlabs/lyra/select";
import { Text } from "@synnaxlabs/lyra/text";
import { Eraser, Panel as PPanel } from "@synnaxlabs/pluto";
import { destructor, search } from "@synnaxlabs/x";
import {
  type KeyboardEvent,
  type ReactElement,
  useEffect,
  useRef,
  useState,
} from "react";

import { CSS } from "@/platform/css";
import { Panel } from "@/platform/panel";
import { Item } from "@/platform/selector/Item";
import { type Group, type Section, type Selectable } from "@/platform/selector/types";

export interface CreateParams {
  sections: Section[];
  tabTitle: string;
  text: string;
  icon: Icon.ReactElement;
  searchPlaceholder: string;
}

interface Entry {
  selectable: Selectable;
  section: Section;
  group?: Group;
}

const entries = (sections: Section[]): Entry[] =>
  sections.flatMap((section) => [
    ...(section.selectables ?? []).map((selectable) => ({ selectable, section })),
    ...(section.groups ?? []).flatMap((group) =>
      group.selectables.map((selectable) => ({ selectable, section, group })),
    ),
  ]);

const haystack = ({ selectable, section, group }: Entry): string =>
  [selectable.title, group?.title ?? "", section.title ?? ""].join(" ");

interface Shown {
  sections: Section[];
  /** The function each item's useOnSelect returned, by item type. */
  handlers: Map<string, () => void>;
}

/**
 * Calls the hooks of every section and item, and leaves out the sections, groups, and
 * items whose useVisible returns false.
 */
const useShown = (sections: Section[], tabKey?: panel.TabKey): Shown => {
  "use no memo"; // Calls a hook per section and item; sections is a fixed module list.
  const all = entries(sections).map((e) => e.selectable);
  const handlers = new Map(all.map((s) => [s.type, s.useOnSelect({ tabKey })]));
  const visible = new Set(
    all.filter((s) => s.useVisible?.() ?? true).map((s) => s.type),
  );
  const sectionsVisible = sections.map((s) => s.useVisible?.() ?? true);
  const isVisible = (s: Selectable): boolean => visible.has(s.type);
  const shown = sections
    .filter((_, i) => sectionsVisible[i])
    .map((section) => ({
      ...section,
      selectables: section.selectables?.filter(isVisible),
      groups: section.groups
        ?.map((group) => ({
          ...group,
          selectables: group.selectables.filter(isVisible),
        }))
        .filter((group) => group.selectables.length > 0),
    }))
    .filter((section) => entries([section]).length > 0);
  return { sections: shown, handlers };
};

const NO_SUBSCRIBE = (): destructor.Destructor => destructor.NOOP;

/** Spans as many grid columns as the section has cards. */
const sectionSpan = (section: Section): PCSS.VarProperties => ({
  "--pluto-section-span": section.selectables?.length ?? 1,
});

interface BodyProps extends Omit<CreateParams, "tabTitle"> {
  tabKey?: panel.TabKey;
}

const Body = ({
  text,
  icon,
  sections,
  tabKey,
  searchPlaceholder,
}: BodyProps): ReactElement => {
  const [query, setQuery] = useState("");
  // The arrow keys and Enter act only while the search has focus, so the cursor shows
  // only then.
  const [focused, setFocused] = useState(false);
  const rootRef = useRef<HTMLDivElement>(null);
  const searchRef = useRef<HTMLInputElement>(null);
  const attached = Portal.useAttached();
  useEffect(() => {
    if (attached) searchRef.current?.focus();
  }, [attached]);
  const { sections: shown, handlers } = useShown(sections, tabKey);
  const primary = shown.filter((s) => s.selectables != null);
  const catalog = shown.filter((s) => s.groups != null);
  const order = entries([...primary, ...catalog])
    .filter((e) => search.matches(haystack(e), query))
    .map((e) => e.selectable.type);
  const matched = new Set(order);
  const searching = query.trim() !== "";
  const find = (type: string): HTMLElement | null | undefined =>
    rootRef.current?.querySelector<HTMLElement>(`[data-selectable="${type}"]`);
  const { hover, setHover } = Select.useHover({
    getOrder: () => order,
    subscribe: NO_SUBSCRIBE,
    initialHover: searching ? 0 : -1,
    click: (type) => find(type)?.click(),
    scrollTo: (type) => find(type)?.scrollIntoView({ block: "nearest" }),
    enableTriggers: () => document.activeElement === searchRef.current,
  });
  const dimmed = (selectables: Selectable[]): boolean =>
    searching && !selectables.some((s) => matched.has(s.type));
  const sectionClass = (section: Section): string =>
    CSS.cls(
      CSS.BE("layout-selector", "section"),
      dimmed(entries([section]).map((e) => e.selectable)) && CSS.M("dimmed"),
    );
  const item = (selectable: Selectable, variant: "card" | "action") => {
    const onSelect = handlers.get(selectable.type);
    if (onSelect == null)
      throw new Error(`no handler for selectable ${selectable.type}`);
    return (
      <Item
        key={selectable.type}
        selectable={selectable}
        variant={variant}
        disabled={!matched.has(selectable.type)}
        hovered={focused && hover === selectable.type}
        onSelect={onSelect}
        onHover={setHover}
      />
    );
  };
  const heading = (section: Section) =>
    section.title != null && (
      <Text.Text
        level="small"
        color={9}
        className={CSS.BE("layout-selector", "heading")}
      >
        {section.title}
      </Text.Text>
    );

  const handleKeyDown = (e: KeyboardEvent<HTMLInputElement>): void => {
    if (e.key !== "Escape" || query === "") return;
    e.preventDefault();
    e.stopPropagation();
    setQuery("");
  };

  return (
    <Flex.Box
      y
      className={CSS.cls(
        CSS.BE("layout-selector", "body"),
        focused && CSS.M("focused"),
      )}
      ref={rootRef}
    >
      <div className={CSS.BE("layout-selector", "header")}>
        <Text.Text
          level="h4"
          color={10}
          weight={450}
          className={CSS.BE("layout-selector", "title")}
        >
          {icon}
          {text}
        </Text.Text>
        <Input.Text
          ref={searchRef}
          value={query}
          onChange={setQuery}
          onKeyDown={handleKeyDown}
          onFocus={() => setFocused(true)}
          onBlur={() => setFocused(false)}
          placeholder={searchPlaceholder}
          startContent={<Icon.Search color={8} />}
          className={CSS.BE("layout-selector", "search")}
          size="huge"
          variant="text"
          full="x"
        />
      </div>
      {primary.length > 0 && (
        <div className={CSS.BE("layout-selector", "primary")}>
          {primary.map((section) => (
            <div
              key={section.key}
              className={sectionClass(section)}
              style={sectionSpan(section)}
            >
              {heading(section)}
              {section.selectables?.map((s) => item(s, "card"))}
            </div>
          ))}
        </div>
      )}
      {catalog.map((section) => (
        <div key={section.key} className={sectionClass(section)}>
          {heading(section)}
          <div className={CSS.BE("layout-selector", "groups")}>
            {section.groups?.map((group) => (
              <div
                key={group.key}
                className={CSS.cls(
                  CSS.BE("layout-selector", "group"),
                  dimmed(group.selectables) && CSS.M("dimmed"),
                )}
              >
                <Text.Text
                  level="p"
                  color={10}
                  className={CSS.BE("layout-selector", "group-name")}
                  overflow="fade"
                >
                  {group.icon}
                  {group.title}
                </Text.Text>
                <div className={CSS.BE("layout-selector", "actions")}>
                  {group.selectables.map((s) => item(s, "action"))}
                </div>
              </div>
            ))}
          </div>
        </div>
      ))}
    </Flex.Box>
  );
};

/**
 * Creates a tab that offers the selectables of each section. A search disables the
 * items that do not match it, and the arrow keys and Enter pick a match.
 */
export const create = ({ tabTitle, icon, ...rest }: CreateParams): Panel.Tab => {
  const TabIcon: Panel.TabIcon = () => icon;
  const Content: Panel.Content = (): ReactElement => {
    const tabKey = PPanel.useOptionalTabKey();
    return (
      <Eraser.Eraser>
        <Flex.Box
          className={CSS.BE("layout-selector", "frame")}
          align="center"
          grow
          full
        >
          <Body icon={icon} tabKey={tabKey} {...rest} />
        </Flex.Box>
      </Eraser.Eraser>
    );
  };
  Content.displayName = `${tabTitle}.Selector`;
  const Name = Panel.createStaticTabName({ name: tabTitle, icon });
  return { Content, Name, Icon: TabIcon };
};
