// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/log/Log.css";

import { box, location, strings } from "@synnaxlabs/x";
import { type ReactElement, type ReactNode, useCallback, useRef } from "react";

import { Button } from "@/button";
import { CSS } from "@/css";
import { type Flex } from "@/flex";
import { useCombinedRefs } from "@/hooks/ref";
import { Icon } from "@/icon";
import { use, type UseProps } from "@/log/use";
import { Menu } from "@/menu";
import { Status } from "@/status/base";
import { Triggers } from "@/triggers";
import { Canvas } from "@/vis/canvas";

const COPY_TRIGGER: Triggers.Trigger = ["Control", "C"];
const SELECT_ALL_TRIGGER: Triggers.Trigger = ["Control", "A"];
const ESCAPE_TRIGGER: Triggers.Trigger = ["Escape"];
const PAUSE_TRIGGER: Triggers.Trigger = ["H"];

export interface LogProps extends UseProps, Omit<Flex.BoxProps, "color"> {
  emptyContent?: ReactElement;
  extraContextMenuItems?: ReactNode;
}

export const Log = ({
  aetherKey,
  font,
  className,
  visible,
  showChannelNames,
  showReceiptTimestamp,
  timestampPrecision,
  channels,
  emptyContent = (
    <Status.Summary center level="h3" variant="disabled" hideIcon>
      Empty Log
    </Status.Summary>
  ),
  color,
  telem,
  extraContextMenuItems,
  ...rest
}: LogProps): ReactElement | null => {
  const { state, setState } = use({
    aetherKey,
    font,
    visible,
    showChannelNames,
    showReceiptTimestamp,
    timestampPrecision,
    channels,
    color,
    telem,
  });

  const {
    scrolling,
    empty,
    selectedText,
    selectedLines,
    region,
    visibleStart,
    computedLineHeight,
    entryCount,
  } = state;

  const resizeRef = Canvas.useRegion(
    useCallback((b) => setState((s) => ({ ...s, region: b })), [setState]),
  );
  const containerRef = useRef<HTMLDivElement>(null);
  const combinedRef = useCombinedRefs(resizeRef, containerRef);

  const draggingRef = useRef(false);

  const mouseYToEntryIndex = useCallback(
    (clientY: number): number => {
      if (computedLineHeight <= 0) return 0;
      const localY = clientY - box.top(region);
      const lineIndex = Math.floor((localY - 6) / computedLineHeight);
      return Math.max(0, visibleStart + lineIndex);
    },
    [region, computedLineHeight, visibleStart],
  );

  const handleMouseDown = useCallback(
    (e: React.MouseEvent) => {
      if (e.button !== 0) return;
      const idx = mouseYToEntryIndex(e.clientY);
      draggingRef.current = true;
      if (e.shiftKey) setState((s) => ({ ...s, selectionEnd: idx }));
      else
        setState((s) => {
          if (s.selectionStart === idx && s.selectionEnd === idx)
            return { ...s, selectionStart: -1, selectionEnd: -1 };
          return { ...s, selectionStart: idx, selectionEnd: idx };
        });
    },
    [mouseYToEntryIndex, setState],
  );

  const handleMouseMove = useCallback(
    (e: React.MouseEvent) => {
      if (!draggingRef.current) return;
      const idx = mouseYToEntryIndex(e.clientY);
      setState((s) => ({ ...s, selectionEnd: idx }));
    },
    [mouseYToEntryIndex, setState],
  );

  const handleMouseUp = useCallback(() => {
    draggingRef.current = false;
  }, []);

  const buildCopyHTML = useCallback((): string => {
    const lines = selectedLines.map((l) => {
      const escaped = strings.escapeHTML(l.text);
      if (l.color.length === 0) return escaped;
      // Preserve color and font when pasting into rich text editors.
      return `<span style="color: ${strings.escapeHTML(l.color)}">${escaped}</span>`;
    });
    return `<pre style="font-family: monospace">${lines.join("\n")}</pre>`;
  }, [selectedLines]);

  const addStatus = Status.useAdder();
  const notifyCopied = useCallback(
    (count: number) =>
      addStatus({
        variant: "success",
        message: `Copied ${count} ${count === 1 ? "line" : "lines"} to clipboard`,
      }),
    [addStatus],
  );

  const copyToClipboard = useCallback(() => {
    if (selectedText.length === 0) return;
    const item = new ClipboardItem({
      "text/html": new Blob([buildCopyHTML()], { type: "text/html" }),
      "text/plain": new Blob([selectedText], { type: "text/plain" }),
    });
    const count = selectedLines.length;
    void navigator.clipboard.write([item]).then(() => notifyCopied(count));
  }, [selectedText, selectedLines.length, buildCopyHTML, notifyCopied]);

  Triggers.use({
    triggers: [ESCAPE_TRIGGER],
    callback: useCallback(
      ({ stage }: Triggers.UseEvent) => {
        if (stage !== "start") return;
        setState((s) => ({
          ...s,
          selectionStart: -1,
          selectionEnd: -1,
          selectedText: "",
        }));
      },
      [setState],
    ),
  });

  Triggers.use({
    triggers: [PAUSE_TRIGGER],
    region: containerRef,
    callback: useCallback(
      ({ stage }: Triggers.UseEvent) => {
        if (stage !== "start") return;
        setState((s) => ({ ...s, scrolling: !s.scrolling }));
      },
      [setState],
    ),
  });

  Triggers.use({
    triggers: [SELECT_ALL_TRIGGER],
    callback: useCallback(
      ({ stage }: Triggers.UseEvent) => {
        if (stage !== "start" || entryCount === 0) return;
        setState((s) => ({
          ...s,
          selectionStart: 0,
          selectionEnd: entryCount - 1,
        }));
      },
      [entryCount, setState],
    ),
  });

  const { className: menuClassName, ...menuProps } = Menu.useContextMenu();
  const hasSelection = selectedText.length > 0;

  const handleMenuSelect = useCallback(
    (key: string) => {
      if (key === "copy") copyToClipboard();
    },
    [copyToClipboard],
  );

  const menuContent = useCallback(
    () => (
      <Menu.Menu level="small" onChange={handleMenuSelect}>
        <Menu.Item
          itemKey="copy"
          trigger={COPY_TRIGGER}
          triggerIndicator
          disabled={!hasSelection}
        >
          <Icon.Copy />
          Copy
        </Menu.Item>
        {extraContextMenuItems != null && (
          <>
            <Menu.Divider />
            {extraContextMenuItems}
          </>
        )}
      </Menu.Menu>
    ),
    [handleMenuSelect, hasSelection, extraContextMenuItems],
  );

  return (
    <Menu.ContextMenu className={menuClassName} menu={menuContent} {...menuProps}>
      <div
        ref={combinedRef}
        tabIndex={0}
        className={CSS(CSS.B("log"), className)}
        onWheel={(e) => {
          setState((s) => ({
            ...s,
            wheelPos: s.wheelPos - e.deltaY,
            scrolling: s.scrolling ? s.scrolling : e.deltaY < 0,
          }));
        }}
        onMouseDown={handleMouseDown}
        onMouseMove={handleMouseMove}
        onMouseUp={handleMouseUp}
        onCopy={(e) => {
          if (selectedText.length === 0) return;
          e.preventDefault();
          e.clipboardData.setData("text/plain", selectedText);
          e.clipboardData.setData("text/html", buildCopyHTML());
          notifyCopied(selectedLines.length);
        }}
        onContextMenu={menuProps.open}
        {...rest}
      >
        {empty ? (
          emptyContent
        ) : (
          <Button.Button
            className={CSS(CSS.BE("log", "live"), scrolling && CSS.M("active"))}
            variant="outlined"
            onClick={() => setState((s) => ({ ...s, scrolling: !s.scrolling }))}
            tooltip={scrolling ? "Resume Scrolling" : "Pause Scrolling"}
            tooltipLocation={location.BOTTOM_LEFT}
          >
            <Icon.Dynamic />
          </Button.Button>
        )}
      </div>
    </Menu.ContextMenu>
  );
};
