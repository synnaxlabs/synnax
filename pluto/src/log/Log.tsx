// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/log/Log.css";

import { box, location, type optional } from "@synnaxlabs/x";
import { type ReactElement, useCallback, useEffect, useRef } from "react";
import { type z } from "zod";

import { Aether } from "@/aether";
import { Button } from "@/button";
import { CSS } from "@/css";
import { type Flex } from "@/flex";
import { Icon } from "@/icon";
import { log } from "@/log/aether";
import { useMemoDeepEqual } from "@/memo";
import { Menu } from "@/menu";
import { Status } from "@/status/base";
import { type Triggers } from "@/triggers";
import { Canvas } from "@/vis/canvas";

const COPY_FLASH_DURATION_MS = 150;
const COPY_TRIGGER: Triggers.Trigger = ["Control", "C"];

export interface LogProps
  extends
    optional.Optional<
      Omit<
        z.input<typeof log.logState>,
        | "region"
        | "scrollPosition"
        | "scrollback"
        | "empty"
        | "scrolling"
        | "wheelPos"
        | "selectionStart"
        | "selectionEnd"
        | "visibleStart"
        | "selectedText"
        | "selectedLines"
        | "computedLineHeight"
        | "copyFlash"
      >,
      "visible"
    >,
    Omit<Flex.BoxProps, "color">,
    Aether.ComponentProps {
  emptyContent?: ReactElement;
}

export const Log = ({
  aetherKey,
  font,
  className,
  visible = true,
  showChannelNames = true,
  timestampPrecision = 0,
  channelConfigs = {},
  channels = [],
  emptyContent = (
    <Status.Summary center level="h3" variant="disabled" hideIcon>
      Empty Log
    </Status.Summary>
  ),
  color,
  telem,
  ...rest
}: LogProps): ReactElement | null => {
  const memoProps = useMemoDeepEqual({
    font,
    color,
    telem,
    visible,
    showChannelNames,
    timestampPrecision,
    channelConfigs,
    channels,
  });
  const [, state, setState] = Aether.use({
    type: log.Log.TYPE,
    schema: log.logState,
    initialState: {
      empty: true,
      region: box.ZERO,
      scrolling: false,
      wheelPos: 0,
      ...memoProps,
    },
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

  useEffect(() => {
    setState((s) => ({ ...s, ...memoProps }));
  }, [memoProps, setState]);

  const resizeRef = Canvas.useRegion(
    useCallback((b) => setState((s) => ({ ...s, region: b })), [setState]),
  );

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

  const copyToClipboard = useCallback(() => {
    if (selectedText.length === 0) return;
    const escapeHTML = (s: string): string =>
      s
        .replace(/&/g, "&amp;")
        .replace(/</g, "&lt;")
        .replace(/>/g, "&gt;")
        .replace(/"/g, "&quot;")
        .replace(/'/g, "&#39;");
    const lines = selectedLines.map((l) => {
      const escaped = escapeHTML(l.text);
      if (l.color.length === 0) return escaped;
      return `<span style="color: ${escapeHTML(l.color)}">${escaped}</span>`;
    });
    const html = `<pre style="font-family: monospace">${lines.join("\n")}</pre>`;
    const item = new ClipboardItem({
      "text/html": new Blob([html], { type: "text/html" }),
      "text/plain": new Blob([selectedText], { type: "text/plain" }),
    });
    void navigator.clipboard.write([item]).then(() => {
      setState((s) => ({ ...s, copyFlash: true }));
      setTimeout(
        () => setState((s) => ({ ...s, copyFlash: false })),
        COPY_FLASH_DURATION_MS,
      );
    });
  }, [selectedText, selectedLines, setState]);

  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent): void => {
      if (e.key === "Escape") {
        setState((s) => ({
          ...s,
          selectionStart: -1,
          selectionEnd: -1,
          selectedText: "",
        }));
        return;
      }
      if (!(e.metaKey || e.ctrlKey)) return;
      if (e.key === "c" && selectedText.length > 0) {
        e.preventDefault();
        copyToClipboard();
      } else if (e.key === "a" && entryCount > 0) {
        // Select All
        e.preventDefault();
        setState((s) => ({
          ...s,
          selectionStart: 0,
          selectionEnd: entryCount - 1,
        }));
      }
    };
    window.addEventListener("keydown", handleKeyDown);
    return () => window.removeEventListener("keydown", handleKeyDown);
  }, [selectedText, entryCount, setState, copyToClipboard]);

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
      </Menu.Menu>
    ),
    [handleMenuSelect, hasSelection],
  );

  return (
    <Menu.ContextMenu className={menuClassName} menu={menuContent} {...menuProps}>
      <div
        ref={resizeRef}
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
