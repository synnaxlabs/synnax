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

// Select-all sends "to end" rather than a concrete index so the entry count never
// has to be synced from the worker on every batch. The worker holds entries.length
// and its slice/clamp logic resolves this to the true last entry at render time.
const SELECT_ALL_END = Number.MAX_SAFE_INTEGER;

type Mode = "selectAll" | "clearSelection" | "togglePause" | "default";

const TRIGGER_CONFIG: Triggers.ModeConfig<Mode> = {
  selectAll: [["Control", "A"]],
  clearSelection: [["Escape"]],
  togglePause: [["H"]],
  default: [],
  defaultMode: "default",
};

const FLATTENED_TRIGGERS = Triggers.flattenConfig(TRIGGER_CONFIG);

export interface BaseProps extends UseProps, Omit<Flex.BoxProps, "color"> {
  emptyContent?: ReactElement;
  extraContextMenuItems?: ReactNode;
  enableTriggers?: boolean | (() => boolean);
}

export const Base = ({
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
  enableTriggers,
  ...rest
}: BaseProps): ReactElement | null => {
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
    triggers: FLATTENED_TRIGGERS,
    callback: useCallback(
      ({ triggers, stage }: Triggers.UseEvent) => {
        if (stage !== "start") return;
        if (enableTriggers === false) return;
        if (typeof enableTriggers === "function" && !enableTriggers()) return;
        const mode = Triggers.determineMode(TRIGGER_CONFIG, triggers);
        if (mode === "selectAll")
          setState((s) => ({ ...s, selectionStart: 0, selectionEnd: SELECT_ALL_END }));
        else if (mode === "clearSelection")
          setState((s) => ({
            ...s,
            selectionStart: -1,
            selectionEnd: -1,
            selectedText: "",
          }));
        else if (mode === "togglePause")
          setState((s) => ({ ...s, scrolling: !s.scrolling }));
      },
      [setState, enableTriggers],
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
