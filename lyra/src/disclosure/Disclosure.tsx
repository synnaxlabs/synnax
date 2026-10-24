// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/disclosure/Disclosure.css";
import "@/pane/Pane.css";

import { type ComponentPropsWithoutRef, type ReactElement } from "react";

import { CSS } from "@/css";
import { Icon } from "@/icon";

/** Props for {@link Disclosure}. */
export interface DisclosureProps extends ComponentPropsWithoutRef<"details"> {}

/**
 * A pane that opens and closes under its {@link Summary}. It is a native `<details>`
 * element, so it works with no JavaScript, and find in page opens it.
 *
 * @example
 * <Disclosure.Disclosure>
 *   <Disclosure.Summary>Not seeing your hardware?</Disclosure.Summary>
 *   <Disclosure.Content>{checks}</Disclosure.Content>
 * </Disclosure.Disclosure>
 */
export const Disclosure = ({ className, ...rest }: DisclosureProps): ReactElement => (
  <details
    className={CSS.cls(CSS.B("pane"), CSS.B("disclosure"), className)}
    {...rest}
  />
);

/** Props for {@link Summary}. */
export interface SummaryProps extends ComponentPropsWithoutRef<"summary"> {}

/** The bar that opens and closes the section. It ends with a caret. */
export const Summary = ({
  className,
  children,
  ...rest
}: SummaryProps): ReactElement => (
  <summary
    className={CSS.cls(
      CSS.BE("pane", "bar"),
      CSS.BE("disclosure", "summary"),
      className,
    )}
    {...rest}
  >
    <span className={CSS.BE("disclosure", "title")}>{children}</span>
    <Icon.Caret.Right className={CSS.BE("disclosure", "indicator")} />
  </summary>
);

/** Props for {@link Content}. */
export interface ContentProps extends ComponentPropsWithoutRef<"div"> {}

/** The body that shows while the section is open. */
export const Content = ({ className, ...rest }: ContentProps): ReactElement => (
  <div className={CSS.cls(CSS.BE("disclosure", "content"), className)} {...rest} />
);
