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
   * focus on the element that opened the view.
   */
  open: (title: string, content: ReactNode) => void;
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
  useEffect(() => {
    if (view != null || opener.current == null) return;
    if (opener.current.isConnected) opener.current.focus();
    else root.current?.focus();
    opener.current = null;
  }, [view]);
  const open = (title: string, content: ReactNode): void => {
    opener.current =
      document.activeElement instanceof HTMLElement ? document.activeElement : null;
    const width = root.current?.getBoundingClientRect().width ?? 0;
    setView({ title, content, width });
    onViewingChange?.(true);
  };
  const back = (): void => {
    setView(null);
    onViewingChange?.(false);
  };
  return (
    <ViewsContext value={{ open, viewing: view != null }}>
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
                onClick={back}
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
