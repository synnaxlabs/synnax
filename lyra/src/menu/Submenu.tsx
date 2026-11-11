// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/menu/Submenu.css";

import { location } from "@synnaxlabs/x";
import { type ReactElement, type RefCallback } from "react";

import { Button } from "@/button";
import { context } from "@/context";
import { CSS } from "@/css";
import { Dialog } from "@/dialog";
import { Flex } from "@/flex";
import { useCombinedRefs } from "@/hooks";
import { Icon } from "@/icon";
import { useContext } from "@/menu/Menu";
import { useControllerContext } from "@/menu/useSubmenus";
import { type position } from "@/position";
import { Text } from "@/text";

interface ContextValue {
  itemKey: string;
  /** Whether the submenu replaced another open submenu. */
  swap: boolean;
  /** Holds the surface the submenu draws, which the pointer aims at. */
  surfaceRef: RefCallback<HTMLElement>;
}

export const [SubmenuContext, useSubmenuContext] = context.create<ContextValue>({
  displayName: "Menu.SubmenuContext",
  providerName: "Menu.Submenu",
});

const BESIDE: position.LocationPreference = {
  targetCorner: location.TOP_RIGHT,
  dialogCorner: location.TOP_LEFT,
};

/** Props for {@link Submenu}. */
export interface SubmenuProps extends Omit<
  Dialog.FrameProps,
  "visible" | "onVisibleChange" | "location" | "variant"
> {
  /** Names the submenu among the others of its menu. */
  itemKey: string;
}

/**
 * A submenu of a `Menu.Menu`: a {@link SubmenuTrigger} and the `Menu.Dialog` it opens
 * beside the menu. One submenu of a menu is open at a time. It opens 100 ms after the
 * pointer stops on its trigger, or at once on a click or ArrowRight. While the pointer
 * moves toward it, the items it crosses do not light or open. A rest on an item with no
 * submenu closes it. ArrowLeft and Escape close it. A `Select.Frame` in it can hold the
 * trigger, with a `Select.Dialog` on the menu surface.
 *
 * @example
 * <Menu.Submenu itemKey="stage">
 *   <Menu.SubmenuTrigger><Icon.ToDo />Stage</Menu.SubmenuTrigger>
 *   <Menu.Dialog>{stages}</Menu.Dialog>
 * </Menu.Submenu>
 */
export const Submenu = ({ itemKey, ...rest }: SubmenuProps): ReactElement => {
  const { open, swap, setVisible, surfaceRef } = useControllerContext("Menu.Submenu");
  const ctxValue = { itemKey, swap, surfaceRef: surfaceRef(itemKey) };
  return (
    <SubmenuContext value={ctxValue}>
      <Dialog.Frame
        variant="floating"
        location={BESIDE}
        visible={open === itemKey}
        onVisibleChange={(next) => setVisible(itemKey, next)}
        {...rest}
      />
    </SubmenuContext>
  );
};

/** Props for {@link SubmenuTrigger}. */
export interface SubmenuTriggerProps extends Omit<
  Button.ButtonProps,
  "onClick" | "onKeyDown" | "onPointerEnter" | "onPointerMove" | "onPointerLeave"
> {}

/**
 * The menu item that opens its {@link Submenu}. It stays lit while the submenu is open,
 * and ends with a caret that points at the submenu.
 */
export const SubmenuTrigger = ({
  ref,
  className,
  children,
  ...rest
}: SubmenuTriggerProps): ReactElement => {
  const { itemKey } = useSubmenuContext("Menu.SubmenuTrigger");
  const { trigger } = useControllerContext("Menu.SubmenuTrigger");
  const { level = "p", gap } = useContext();
  const { triggerRef, visible } = Dialog.useContext();
  const { ref: handlerRef, ...handlers } = trigger(itemKey);
  const combinedRef = useCombinedRefs(ref, triggerRef, handlerRef);
  return (
    <Button.Button
      ref={combinedRef}
      variant="text"
      level={level}
      gap={gap}
      overflow="nowrap"
      justify="between"
      full="x"
      role="menuitem"
      aria-haspopup="dialog"
      aria-expanded={visible}
      className={CSS.cls(CSS.B("menu-item"), CSS.BM("menu-item", "submenu"), className)}
      {...handlers}
      {...rest}
    >
      <Flex.Box x align="center" gap={gap} el="span">
        {children}
      </Flex.Box>
      <Text.Text el="span" color={8}>
        <Icon.Caret.Right />
      </Text.Text>
    </Button.Button>
  );
};
