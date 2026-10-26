// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/pane/Pane.css";

import { type ComponentPropsWithoutRef, type ReactElement } from "react";

import { Button as BaseButton } from "@/button";
import { CSS } from "@/css";
import { Flex } from "@/flex";
import { Header as BaseHeader } from "@/header";
import { Icon } from "@/icon";

/** Props for {@link Pane}. */
export interface PaneProps extends Flex.BoxProps {}

/**
 * A bordered surface that groups related content, with hairlines between its items.
 *
 * @example
 * <Pane.Pane>
 *   <Pane.Header level="p">
 *     <Header.Title>Node 1</Header.Title>
 *   </Pane.Header>
 *   <Pane.Item>T7-Pro</Pane.Item>
 * </Pane.Pane>
 */
export const Pane = ({ className, ...rest }: PaneProps): ReactElement => (
  <Flex.Box y empty className={CSS.cls(CSS.B("pane"), className)} {...rest} />
);

/** Props for {@link Header}. */
export interface HeaderProps extends BaseHeader.HeaderProps {}

/** The tinted bar at the top of a pane. */
export const Header = ({ className, ...rest }: HeaderProps): ReactElement => (
  <BaseHeader.Header className={CSS.cls(CSS.BE("pane", "bar"), className)} {...rest} />
);

/** Props for {@link Disclosure}. */
export interface DisclosureProps extends ComponentPropsWithoutRef<"details"> {}

/**
 * A pane that opens and closes under its {@link Summary}. It is a native `<details>`
 * element, so it works with no JavaScript, and find in page opens it.
 *
 * @example
 * <Pane.Disclosure>
 *   <Pane.Summary>Not seeing your hardware?</Pane.Summary>
 *   <Pane.Item>Check the cable.</Pane.Item>
 * </Pane.Disclosure>
 */
export const Disclosure = ({ className, ...rest }: DisclosureProps): ReactElement => (
  <details
    className={CSS.cls(CSS.B("pane"), CSS.BM("pane", "disclosure"), className)}
    {...rest}
  />
);

/** Props for {@link Summary}. */
export interface SummaryProps extends ComponentPropsWithoutRef<"summary"> {}

/** The bar that opens and closes a {@link Disclosure}. It ends with a caret. */
export const Summary = ({
  className,
  children,
  ...rest
}: SummaryProps): ReactElement => (
  <summary
    className={CSS.cls(CSS.BE("pane", "bar"), CSS.BE("pane", "summary"), className)}
    {...rest}
  >
    <span className={CSS.BE("pane", "title")}>{children}</span>
    <Icon.Caret.Right className={CSS.BE("pane", "indicator")} />
  </summary>
);

/** Props for {@link Item}. */
export interface ItemProps extends Flex.BoxProps {}

/** One item of content in a pane. */
export const Item = ({ className, ...rest }: ItemProps): ReactElement => (
  <Flex.Box
    x
    align="center"
    gap="medium"
    className={CSS.cls(CSS.BE("pane", "item"), className)}
    {...rest}
  />
);

/** Props for {@link Button}. */
export interface ButtonProps extends Omit<BaseButton.ButtonProps, "variant"> {}

/** An item that opens what it names. It ends with a caret that moves on hover. */
export const Button = ({ className, children, ...rest }: ButtonProps): ReactElement => (
  <BaseButton.Button
    variant="text"
    className={CSS.cls(CSS.BE("pane", "item"), CSS.BE("pane", "button"), className)}
    {...rest}
  >
    {children}
    <Icon.Caret.Right className={CSS.BE("pane", "caret")} />
  </BaseButton.Button>
);
