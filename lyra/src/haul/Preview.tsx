// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/haul/Preview.css";

import { Fragment, type ReactElement, type ReactNode } from "react";

import { CSS } from "@/css";
import { Text } from "@/text";

/** Props for {@link Fact}. */
export interface FactProps {
  icon?: ReactNode;
  children: ReactNode;
}

/** A fact that names another object, with that object's icon beside its name. */
export const Fact = ({ icon, children }: FactProps): ReactElement => (
  <span className={CSS.BE("haul-preview", "fact")}>
    {icon}
    {children}
  </span>
);

/** Props for {@link Preview}. */
export interface PreviewProps {
  icon?: ReactNode;
  name: ReactNode;
  /** Short facts about the object, such as its data type, under the name. */
  facts?: ReactNode[];
  /** A color that fills the tile, for objects that are a color. */
  swatch?: string;
  /** How many objects the drag carries. Above one, the preview stacks with a count. */
  count?: number;
}

// Fades a name only when the preview cuts it.
const markCut = (el: HTMLElement | null): void => {
  el?.classList.toggle(
    CSS.BEM("haul-preview", "name", "cut"),
    el.scrollWidth > el.clientWidth,
  );
};

/** The image that follows the pointer while an object drags. */
export const Preview = ({
  icon,
  name,
  facts = [],
  swatch,
  count = 1,
}: PreviewProps): ReactElement => (
  <div
    className={CSS.cls(
      CSS.B("haul-preview"),
      count > 1 && CSS.BM("haul-preview", "stack"),
    )}
  >
    <div className={CSS.BE("haul-preview", "main")}>
      {(icon != null || swatch != null) && (
        <span
          className={CSS.cls(
            CSS.BE("haul-preview", "tile"),
            swatch != null && CSS.BEM("haul-preview", "tile", "swatch"),
          )}
          style={swatch != null ? { background: swatch } : undefined}
        >
          {swatch == null && icon}
        </span>
      )}
      <div className={CSS.BE("haul-preview", "text")}>
        <Text.Text level="p" className={CSS.BE("haul-preview", "name")} ref={markCut}>
          {name}
        </Text.Text>
        {facts.length > 0 && (
          <Text.Text level="small" className={CSS.BE("haul-preview", "facts")}>
            {facts.map((fact, i) => (
              <Fragment key={i}>
                {i > 0 && <span className={CSS.BE("haul-preview", "sep")}>·</span>}
                {fact}
              </Fragment>
            ))}
          </Text.Text>
        )}
      </div>
    </div>
    {count > 1 && <span className={CSS.BE("haul-preview", "count")}>{count}</span>}
  </div>
);

/** Marks a part, such as a close button, that a {@link Copy} leaves out. */
export const OMIT_ATTRIBUTE = "data-haul-omit";

const OMIT_SELECTOR = `[${OMIT_ATTRIBUTE}]`;

/** Props for {@link Copy}. */
export interface CopyProps {
  /** The element to copy, usually the drag's `currentTarget`. */
  of: Element;
}

// Stand-ins for the source's ancestors keep the selectors that depend on them matching.
// They draw nothing. Nothing hovers or presses the copy, so it shows its resting face.
const wrapInAncestors = (source: Element, copy: Element): Element => {
  let wrapped = copy;
  let el = source.parentElement;
  for (; el != null && el !== document.body; el = el.parentElement) {
    const standIn = document.createElement("div");
    standIn.setAttribute("class", el.getAttribute("class") ?? "");
    standIn.style.display = "contents";
    standIn.append(wrapped);
    wrapped = standIn;
  }
  return wrapped;
};

/** A drag preview that copies an element, such as a tab, as it looks at rest. */
export const Copy = ({ of }: CopyProps): ReactElement => {
  const handleRef = (el: HTMLDivElement | null): void => {
    if (el == null) return;
    const copy = of.cloneNode(true) as HTMLElement;
    copy.querySelectorAll(OMIT_SELECTOR).forEach((part) => part.remove());
    Object.assign(copy.style, { margin: "0", transform: "none", opacity: "1" });
    el.replaceChildren(wrapInAncestors(of, copy));
  };
  return <div ref={handleRef} className={CSS.B("haul-copy")} />;
};
