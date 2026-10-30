// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

const NAV = ".page-nav";
const TOGGLE = "[data-page-nav-toggle]";
const SECTION = ".page-nav-section-header";
const TOOLTIP = ".page-nav__tooltip";
const CLOSING_CLASS = "lyra--closing";
/** The browser storage key that remembers a collapsed sidebar. */
export const STORAGE_KEY = "docs-page-nav-collapsed";
/** The key that toggles the sidebar together with Cmd or Ctrl. */
export const SHORTCUT = "B";
// Lyra's tooltip timing and offset, so the sidebar matches the Console.
const DELAY_MS = 700;
const SKIP_DELAY_MS = 300;
const CLOSE_MS = 150;
const OFFSET = 6;

// The flag is on the root, so a script in the head can set it before the body parses.
const collapsed = (): boolean =>
  document.documentElement.dataset.pageNavCollapsed != null;

const sync = (nav: HTMLElement): void => {
  const toggle = nav.querySelector(TOGGLE);
  toggle?.setAttribute("aria-label", toggleLabel());
  toggle?.setAttribute("aria-expanded", String(!collapsed()));
  // Only the collapsed section icons act as links, so only they take focus.
  for (const section of nav.querySelectorAll<HTMLElement>(SECTION))
    section.tabIndex = collapsed() ? 0 : -1;
};

const toggleLabel = (): string => (collapsed() ? "Expand sidebar" : "Collapse sidebar");

const setCollapsed = (nav: HTMLElement, value: boolean): void => {
  if (value) document.documentElement.dataset.pageNavCollapsed = "";
  else delete document.documentElement.dataset.pageNavCollapsed;
  sync(nav);
  try {
    localStorage.setItem(STORAGE_KEY, String(value));
  } catch {
    // Storage is blocked in some private windows; the choice then lasts one page.
  }
};

// A collapsed section icon opens the first page of its section.
const openSection = (section: Element): void => {
  let el = section.nextElementSibling;
  while (el != null && !el.matches(SECTION)) {
    if (el instanceof HTMLAnchorElement) return el.click();
    el = el.nextElementSibling;
  }
};

const stored = (): boolean => {
  try {
    return localStorage.getItem(STORAGE_KEY) === "true";
  } catch {
    return false;
  }
};

const navOf = (target: EventTarget | null): HTMLElement | null =>
  target instanceof Element ? target.closest<HTMLElement>(NAV) : null;

const anchorOf = (target: EventTarget | null): HTMLElement | null => {
  if (!(target instanceof Element)) return null;
  const anchor = target.closest<HTMLElement>(`${TOGGLE}, ${SECTION}`);
  const nav = navOf(anchor);
  if (anchor == null || nav == null) return null;
  if (anchor.matches(SECTION) && !collapsed()) return null;
  return anchor;
};

const editable = (target: EventTarget | null): boolean =>
  target instanceof HTMLElement &&
  (target.isContentEditable || target.closest("input, textarea, select") != null);

/**
 * Starts the collapse control of the reference sidebar. The toggle and Cmd or Ctrl+B
 * collapse the sidebar to its section icons, and the choice persists in the browser.
 * The toggle and each collapsed icon show a Lyra tooltip. Call it once; it follows
 * page swaps by itself.
 */
export const start = (): void => {
  let anchor: HTMLElement | null = null;
  let openTimer: ReturnType<typeof setTimeout> | undefined;
  let closeTimer: ReturnType<typeof setTimeout> | undefined;
  let lastClosed = -Infinity;

  const tooltip = (): HTMLElement | null =>
    document.querySelector<HTMLElement>(TOOLTIP);

  const show = (target: HTMLElement): void => {
    const tip = tooltip();
    const nav = navOf(target);
    if (tip == null || nav == null) return;
    clearTimeout(openTimer);
    clearTimeout(closeTimer);
    const isToggle = target.matches(TOGGLE);
    const label = tip.querySelector("[data-tooltip-label]");
    const keys = tip.querySelector<HTMLElement>("[data-tooltip-keys]");
    if (label != null)
      label.textContent = isToggle
        ? toggleLabel()
        : (target.querySelector(".lyra-text")?.textContent ?? "");
    if (keys != null) keys.hidden = !isToggle;
    anchor?.removeAttribute("aria-describedby");
    anchor = target;
    target.setAttribute("aria-describedby", tip.id);
    tip.classList.remove(CLOSING_CLASS);
    tip.hidden = false;
    const rect = target.getBoundingClientRect();
    const top = rect.top + (rect.height - tip.offsetHeight) / 2;
    // The translate property, since the tooltip's entry animation owns transform.
    tip.style.translate = `${rect.right + OFFSET}px ${top}px`;
  };

  const hide = (immediate = false): void => {
    clearTimeout(openTimer);
    const tip = tooltip();
    if (tip == null || anchor == null) return;
    anchor.removeAttribute("aria-describedby");
    anchor = null;
    lastClosed = performance.now();
    if (immediate) {
      clearTimeout(closeTimer);
      tip.hidden = true;
      return;
    }
    tip.classList.add(CLOSING_CLASS);
    closeTimer = setTimeout(() => {
      tip.hidden = true;
      tip.classList.remove(CLOSING_CLASS);
    }, CLOSE_MS);
  };

  const toggle = (nav: HTMLElement): void => {
    setCollapsed(nav, !collapsed());
    if (anchor != null && anchor.matches(TOGGLE)) show(anchor);
    else hide(true);
  };

  document.addEventListener("pointerover", (e) => {
    if (e.pointerType === "touch") return;
    const target = anchorOf(e.target);
    if (target == null || target === anchor || target === anchorOf(e.relatedTarget))
      return;
    clearTimeout(openTimer);
    if (anchor != null || performance.now() - lastClosed < SKIP_DELAY_MS)
      return show(target);
    openTimer = setTimeout(() => show(target), DELAY_MS);
  });
  document.addEventListener("pointerout", (e) => {
    const target = anchorOf(e.target);
    if (target == null || target === anchorOf(e.relatedTarget)) return;
    if (target === anchor) hide();
    else clearTimeout(openTimer);
  });
  document.addEventListener("pointerdown", () => hide(true));
  document.addEventListener("focusin", (e) => {
    const target = anchorOf(e.target);
    if (target != null && target.matches(":focus-visible")) show(target);
  });
  document.addEventListener("focusout", (e) => {
    if (anchorOf(e.target) === anchor) hide(true);
  });
  document.addEventListener("scroll", () => hide(true), {
    capture: true,
    passive: true,
  });
  document.addEventListener("click", (e) => {
    if (!(e.target instanceof Element)) return;
    const nav = navOf(e.target);
    if (nav == null) return;
    if (e.target.closest(TOGGLE) != null) return toggle(nav);
    const section = e.target.closest(SECTION);
    if (section != null && collapsed()) openSection(section);
  });
  document.addEventListener("keydown", (e) => {
    if (e.key === "Escape") return hide(true);
    const section = anchorOf(e.target);
    if (section?.matches(SECTION) && (e.key === "Enter" || e.key === " ")) {
      e.preventDefault();
      return openSection(section);
    }
    const mod = e.metaKey || e.ctrlKey;
    if (!mod || e.altKey || e.shiftKey || e.code !== `Key${SHORTCUT}`) return;
    if (editable(e.target)) return;
    const nav = document.querySelector<HTMLElement>(NAV);
    // The sidebar is not shown at phone width, so the shortcut leaves the key alone.
    if (nav == null || nav.getClientRects().length === 0) return;
    e.preventDefault();
    toggle(nav);
  });
  // A page swap replaces the root's attributes with the incoming page's, so the
  // incoming root must carry the collapse.
  document.addEventListener("astro:before-swap", (e) => {
    if (e.newDocument.querySelector(NAV) != null && (collapsed() || stored()))
      e.newDocument.documentElement.dataset.pageNavCollapsed = "";
  });
  const load = (): void => {
    hide(true);
    for (const nav of document.querySelectorAll<HTMLElement>(NAV)) sync(nav);
  };
  document.addEventListener("astro:after-swap", load);
  load();
};
