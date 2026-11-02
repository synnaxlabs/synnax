// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { hashTarget } from "@/util/hash";

/**
 * Names the URL query parameter an element syncs with. Tab frames, pickers, and inline
 * values that share a key show the same choice. An empty value marks a tab frame that
 * switches alone.
 */
export const QUERY_ATTRIBUTE = "data-tabs-query";

/** Marks a picker: a menu of options for a query key. */
export const PICKER_ATTRIBUTE = "data-tabs-picker";

/** Dispatched on the document after any element changes its choice. */
export const CHANGE_EVENT = "docs:tabs-change";

const TAB = '[role="tab"]';
const STRIP = '[role="tablist"]';
const PANEL = '[role="tabpanel"]';
const OPTION = "[data-tab-key]";
const SELECTED_CLASS = "lyra--selected";
const CLIPPED_START_CLASS = "lyra-tabs__selector--clipped-start";
const CLIPPED_END_CLASS = "lyra-tabs__selector--clipped-end";

/** Resolves and reacts to the choice for one query key. */
export interface Query {
  /** Returns the choice to show when the URL names none, or null for the first tab. */
  initial?: () => string | null;
  /** Called each time a choice applies, from the URL, a click, or `initial`. */
  onChange?: (value: string) => void;
}

export type Queries = Record<string, Query>;

const keyOf = (el: Element): string => el.getAttribute(QUERY_ATTRIBUTE) ?? "";

// A tab belongs to its nearest frame, so a frame nested in a panel keeps its own tabs.
const ownTabs = (frame: Element): HTMLElement[] =>
  [...frame.querySelectorAll<HTMLElement>(TAB)].filter(
    (tab) => tab.closest(`[${QUERY_ATTRIBUTE}]`) === frame,
  );

// A strip of tabs is static HTML, so this script, not the tabs component, fades each
// edge that hides tabs and scrolls the selected tab into view.
const clip = (strip: Element): void => {
  const max = strip.scrollWidth - strip.clientWidth;
  strip.classList.toggle(CLIPPED_START_CLASS, max > 1 && strip.scrollLeft > 1);
  strip.classList.toggle(CLIPPED_END_CLASS, max > 1 && strip.scrollLeft < max - 1);
};

const clipAll = (): void => document.querySelectorAll(STRIP).forEach(clip);

const reveal = (tab: Element): void => {
  const strip = tab.closest(STRIP);
  if (strip == null) return;
  const port = strip.getBoundingClientRect();
  const { left, right } = tab.getBoundingClientRect();
  // Centered, a tab clears the fade at each edge, and the strip stops at its ends.
  if (left < port.left || right > port.right)
    strip.scrollLeft += (left + right - port.left - port.right) / 2;
  clip(strip);
};

// The panels of a frame without a query key stay findable in the page while hidden,
// and a match selects their tab. A frame with a key holds a choice that applies to the
// whole site, which a search must not change.
const hide = (frame: Element, panel: HTMLElement, hidden: boolean): void => {
  if (!hidden) panel.removeAttribute("hidden");
  else if (keyOf(frame).length === 0) panel.setAttribute("hidden", "until-found");
  else panel.hidden = true;
};

const selectTab = (frame: Element, value: string): boolean => {
  const tabs = ownTabs(frame);
  if (!tabs.some((tab) => tab.dataset.tabKey === value)) return false;
  for (const tab of tabs) {
    const selected = tab.dataset.tabKey === value;
    tab.setAttribute("aria-selected", String(selected));
    tab.tabIndex = selected ? 0 : -1;
    tab.classList.toggle(SELECTED_CLASS, selected);
    const panel = frame.querySelector<HTMLElement>(
      `[id="${tab.getAttribute("aria-controls")}"]`,
    );
    if (panel != null) hide(frame, panel, !selected);
    if (selected) reveal(tab);
  }
  return true;
};

const selectOption = (picker: Element, value: string): boolean => {
  const options = [...picker.querySelectorAll<HTMLElement>(OPTION)];
  const chosen = options.find((option) => option.dataset.tabKey === value);
  if (chosen == null) return false;
  for (const option of options) {
    const selected = option === chosen;
    option.setAttribute("aria-selected", String(selected));
    option.classList.toggle(SELECTED_CLASS, selected);
  }
  const shown = picker.querySelector("[data-tabs-value]");
  shown?.replaceChildren(...[...chosen.childNodes].map((node) => node.cloneNode(true)));
  return true;
};

/**
 * Starts syncing tab frames and pickers in the page. Call it once, after the page
 * parses; it follows page swaps by itself.
 * @param queries - Every query key an element in the docs can carry.
 * @throws {Error} on page load if an element carries a key missing from queries.
 */
export const start = (queries: Queries): void => {
  const query = (key: string): Query => {
    const found = queries[key];
    if (found == null) throw new Error(`no tab query for key ${key}`);
    return found;
  };

  // Keeps the element the reader clicked in place while blocks above it resize.
  const apply = (key: string, value: string, anchor?: Element): void => {
    const top = anchor?.getBoundingClientRect().top;
    for (const el of document.querySelectorAll(`[${QUERY_ATTRIBUTE}="${key}"]`))
      if (el.hasAttribute(PICKER_ATTRIBUTE)) selectOption(el, value);
      else selectTab(el, value);
    query(key).onChange?.(value);
    if (anchor != null && top != null)
      window.scrollBy(0, anchor.getBoundingClientRect().top - top);
    document.dispatchEvent(new CustomEvent(CHANGE_EVENT));
  };

  const choose = (key: string, value: string, anchor: Element): void => {
    const url = new URL(window.location.href);
    url.searchParams.set(key, value);
    // A choice is not a navigation, so it stays out of the history.
    window.history.replaceState(window.history.state, "", url);
    apply(key, value, anchor);
  };

  const load = (): void => {
    const url = new URL(window.location.href);
    const keys = new Set(
      [...document.querySelectorAll(`[${QUERY_ATTRIBUTE}]`)]
        .map(keyOf)
        .filter((key) => key.length > 0),
    );
    let replaced = false;
    for (const key of keys) {
      const fromURL = url.searchParams.get(key);
      const value = fromURL ?? query(key).initial?.() ?? null;
      if (value == null) continue;
      if (fromURL == null) {
        url.searchParams.set(key, value);
        replaced = true;
      }
      apply(key, value);
    }
    if (replaced) window.history.replaceState(window.history.state, "", url);
    for (const panel of document.querySelectorAll<HTMLElement>(`${PANEL}[hidden]`)) {
      const frame = panel.closest(`[${QUERY_ATTRIBUTE}]`);
      if (frame != null) hide(frame, panel, true);
    }
    revealHashTarget();
    clipAll();
    document.dispatchEvent(new CustomEvent(CHANGE_EVENT));
  };

  const select = (target: HTMLElement): void => {
    const frame = target.closest(`[${QUERY_ATTRIBUTE}]`);
    const value = target.dataset.tabKey;
    if (frame == null || value == null) return;
    const key = keyOf(frame);
    if (key.length > 0) choose(key, value, frame);
    else if (selectTab(frame, value))
      document.dispatchEvent(new CustomEvent(CHANGE_EVENT));
  };

  const selectPanel = (panel: Element): void => {
    const tab = document.getElementById(panel.getAttribute("aria-labelledby") ?? "");
    if (tab != null) select(tab);
  };

  // Selects each tab whose panel holds the URL hash target, outermost first.
  const revealHashTarget = (): HTMLElement | null => {
    const target = hashTarget();
    const panels: Element[] = [];
    for (let p = target?.closest(PANEL); p != null; p = p.parentElement?.closest(PANEL))
      panels.unshift(p);
    panels.forEach(selectPanel);
    return target;
  };

  document.addEventListener("click", (e) => {
    if (!(e.target instanceof Element)) return;
    const tab = e.target.closest<HTMLElement>(`[${QUERY_ATTRIBUTE}] ${TAB}`);
    if (tab != null) return select(tab);
    const option = e.target.closest<HTMLElement>(`[${PICKER_ATTRIBUTE}] ${OPTION}`);
    if (option == null) return;
    select(option);
    option.closest<HTMLElement>("[popover]")?.hidePopover();
  });

  document.addEventListener("keydown", (e) => {
    if (!(e.target instanceof HTMLElement) || !e.target.matches(TAB)) return;
    const frame = e.target.closest(`[${QUERY_ATTRIBUTE}]`);
    if (frame == null) return;
    if (e.key === "Enter" || e.key === " ") {
      e.preventDefault();
      return select(e.target);
    }
    const step = { ArrowRight: 1, ArrowLeft: -1 }[e.key];
    if (step == null) return;
    const tabs = ownTabs(frame);
    const next = tabs[(tabs.indexOf(e.target) + step + tabs.length) % tabs.length];
    e.preventDefault();
    select(next);
    next.focus();
  });

  // A menu opens in the top layer, detached from layout, so it is placed by hand.
  document.addEventListener(
    "beforetoggle",
    (e) => {
      if (!(e instanceof ToggleEvent) || e.newState !== "open") return;
      if (!(e.target instanceof HTMLElement)) return;
      const trigger = e.target
        .closest(`[${PICKER_ATTRIBUTE}]`)
        ?.querySelector("[popovertarget]");
      if (trigger == null) return;
      const { left, bottom, width } = trigger.getBoundingClientRect();
      e.target.style.left = `${left}px`;
      e.target.style.top = `${bottom}px`;
      e.target.style.width = `${width}px`;
    },
    // Toggle events do not bubble.
    { capture: true },
  );

  window.addEventListener("hashchange", () => revealHashTarget()?.scrollIntoView());

  // Find in page fires beforematch on each hidden panel that holds the match.
  document.addEventListener(
    "beforematch",
    (e) => {
      if (e.target instanceof Element && e.target.matches(PANEL)) selectPanel(e.target);
    },
    { capture: true },
  );

  // Scroll events do not bubble.
  document.addEventListener(
    "scroll",
    (e) => {
      if (e.target instanceof Element && e.target.matches(STRIP)) clip(e.target);
    },
    { capture: true, passive: true },
  );
  window.addEventListener("resize", clipAll, { passive: true });

  // After a swap, not on page load, so the new page never shows its first tabs.
  document.addEventListener("astro:after-swap", load);
  load();
};
