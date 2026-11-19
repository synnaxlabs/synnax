// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/dialog/Views.css";

import { type ReactElement, type ReactNode, useEffect, useRef, useState } from "react";

import { Button } from "@/button";
import { context } from "@/context";
import { CSS } from "@/css";
import { Flex } from "@/flex";
import { Icon } from "@/icon";
import { Text } from "@/text";

export interface ViewsContextValue {
  /**
   * Shows the content in place of the children, under a Back button and the title, at
   * the width of the children. Back returns to the children as they were, with the
   * focus on the element that opened the view. A click passes its target as the
   * opener, since a click in WebKit focuses no button.
   */
  open: (title: string, content: ReactNode, opener?: HTMLElement) => void;
  /**
   * Returns to the children, as Back does. A given callback runs once the children
   * show, in place of the return of the focus to the opener, or at once when they show
   * already.
   */
  back: (then?: () => void) => void;
  /** If true, a view shows in place of the children. */
  viewing: boolean;
}

const [ViewsContext, useViewsContext] = context.create<ViewsContextValue>({
  displayName: "Dialog.ViewsContext",
  providerName: "Dialog.Views",
});

/**
 * @returns the views around the caller.
 * @throws {Error} outside a `Dialog.Views`.
 */
export const useViews = (): ViewsContextValue => useViewsContext("Dialog.useViews");

interface View {
  title: string;
  content: ReactNode;
  /** The width of the children when the view opened. */
  width: number;
}

export interface ViewsProps extends Omit<Flex.BoxProps, "ref" | "x" | "y" | "gap"> {
  /** Called with true when a view opens, and with false on Back. */
  onViewingChange?: (viewing: boolean) => void;
}

/**
 * A box that can show a view in place of its children, such as an editor for one of
 * its items. The children stay mounted under the view, so their state and scroll stay.
 */
export const Views = ({
  className,
  style,
  children,
  onViewingChange,
  ...rest
}: ViewsProps): ReactElement => {
  const [view, setView] = useState<View | null>(null);
  const root = useRef<HTMLDivElement>(null);
  const opener = useRef<HTMLElement | null>(null);
  const after = useRef<(() => void) | undefined>(undefined);
  // A ref, so that a callback from a view that closed sees the children show.
  const viewing = useRef(false);
  useEffect(() => {
    if (view != null) return;
    const then = after.current;
    after.current = undefined;
    if (then != null) {
      opener.current = null;
      return then();
    }
    if (opener.current == null) return;
    if (opener.current.isConnected) opener.current.focus();
    else root.current?.focus();
    opener.current = null;
  }, [view]);
  const open = (title: string, content: ReactNode, clicked?: HTMLElement): void => {
    const active = document.activeElement;
    if (!(active instanceof HTMLElement)) opener.current = clicked ?? null;
    // A click in WebKit leaves the focus on the body or on a focusable box around the
    // button.
    else if (clicked != null && active.contains(clicked)) opener.current = clicked;
    else opener.current = active;
    const width = root.current?.getBoundingClientRect().width ?? 0;
    viewing.current = true;
    setView({ title, content, width });
    onViewingChange?.(true);
  };
  const back = (then?: () => void): void => {
    if (!viewing.current) return then?.();
    viewing.current = false;
    after.current = then;
    setView(null);
    onViewingChange?.(false);
  };
  return (
    <ViewsContext value={{ open, back, viewing: view != null }}>
      <Flex.Box
        y
        gap={0}
        ref={root}
        tabIndex={-1}
        className={CSS.cls(
          CSS.B("views"),
          view != null && CSS.BM("views", "viewing"),
          className,
        )}
        style={view == null ? style : { ...style, width: view.width }}
        {...rest}
      >
        {view != null && (
          <>
            <Flex.Box
              x
              align="center"
              gap="small"
              className={CSS.BE("views", "header")}
            >
              <Button.Button
                variant="text"
                size="small"
                square
                textColor={9}
                onClick={() => back()}
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
            <Flex.Box y gap="medium" className={CSS.BE("views", "view")}>
              {view.content}
            </Flex.Box>
          </>
        )}
        {children}
      </Flex.Box>
    </ViewsContext>
  );
};
