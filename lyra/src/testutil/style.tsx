// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type PropsWithChildren, type ReactElement, type ReactNode } from "react";
import { render, type RenderResult } from "vitest-browser-react";

import { Theming } from "@/theming";

const Themed = ({ children }: PropsWithChildren): ReactElement => (
  <Theming.Provider theme={Theming.SYNNAX_LIGHT}>{children}</Theming.Provider>
);

/**
 * Renders the element in a real browser under the light theme, which sets every theme
 * token on the document root, as the Console does.
 */
export const renderThemed = (element: ReactNode): Promise<RenderResult> =>
  render(element, { wrapper: Themed });

/**
 * @returns the computed value of a CSS property set to the value, such as a theme
 * token. Compare it with a computed style, so that a spec pins which token applies and
 * not the token's current value.
 */
export const resolve = (value: string, property: string = "color"): string => {
  const probe = document.createElement("div");
  probe.style.setProperty(property, value);
  document.body.appendChild(probe);
  const computed = getComputedStyle(probe).getPropertyValue(property);
  probe.remove();
  return computed;
};

/** @returns the computed value of a CSS property on the element. */
export const computed = (el: Element, property: string): string =>
  getComputedStyle(el).getPropertyValue(property);

/** @returns the rendered height of the element in pixels. */
export const height = (el: Element): number => el.getBoundingClientRect().height;
