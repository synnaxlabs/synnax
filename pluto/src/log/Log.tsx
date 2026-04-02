// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/log/Log.css";

import { box, location, type optional, strings } from "@synnaxlabs/x";
import { type ReactElement, useCallback, useEffect, useMemo, useRef } from "react";
import { type z } from "zod";

import { Aether } from "@/aether";
import { Button } from "@/button";
import { Channel } from "@/channel";
import { CSS } from "@/css";
import { type Flex } from "@/flex";
import { Icon } from "@/icon";
import { log } from "@/log/aether";
import { useMemoDeepEqual } from "@/memo";
import { Menu } from "@/menu";
import { Status } from "@/status/base";
import { Triggers } from "@/triggers";
import { Canvas } from "@/vis/canvas";


const COPY_FLASH_DURATION_MS = 150;
const COPY_TRIGGER: Triggers.Trigger = ["Control", "C"];
const SELECT_ALL_TRIGGER: Triggers.Trigger = ["Control", "A"];
const ESCAPE_TRIGGER: Triggers.Trigger = ["Escape"];

// Worker-computed fields that the caller should not pass as props.
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
        | "channelNames"
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
  showReceiptTimestamp = true,
  timestampPrecision = 0,
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
  const numericChannels = useMemo(
    () =>
      channels
        .map((e) => e.channel)
        .filter((ch): ch is number => typeof ch === "number" && ch > 0),
    [channels],
  );
  const { data: retrievedChannels } = Channel.useRetrieveMultiple({
    keys: numericChannels,
  });
  const channelNames = useMemo(() => {
    const names: Record<string, string> = {};
    if (retrievedChannels != null)
      for (const ch of retrievedChannels) names[String(ch.key)] = ch.name;
    return names;
  }, [retrievedChannels]);

  const memoProps = useMemoDeepEqual({
    font,
    color,
    telem,
    visible,
    showChannelNames,
    showReceiptTimestamp,
    timestampPrecision,
    channelNames,
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

  const buildCopyHTML = useCallback((): string => {
    const lines = selectedLines.map((l) => {
      const escaped = strings.escapeHTML(l.text);
      if (l.color.length === 0) return escaped;
      // Preserve color and font when pasting into rich text editors.
      return `<span style="color: ${strings.escapeHTML(l.color)}">${escaped}</span>`;
    });
    return `<pre style="font-family: monospace">${lines.join("\n")}</pre>`;
  }, [selectedLines]);

  const flashCopy = useCallback(() => {
    setState((s) => ({ ...s, copyFlash: true }));
    setTimeout(
      () => setState((s) => ({ ...s, copyFlash: false })),
      COPY_FLASH_DURATION_MS,
    );
  }, [setState]);

  const copyToClipboard = useCallback(() => {
    if (selectedText.length === 0) return;
    const item = new ClipboardItem({
      "text/html": new Blob([buildCopyHTML()], { type: "text/html" }),
      "text/plain": new Blob([selectedText], { type: "text/plain" }),
    });
    void navigator.clipboard.write([item]).then(flashCopy);
  }, [selectedText, buildCopyHTML, flashCopy]);

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
      </Menu.Menu>
    ),
    [handleMenuSelect, hasSelection],
  );

  return (
    <Menu.ContextMenu className={menuClassName} menu={menuContent} {...menuProps}>
      <div
        ref={resizeRef}
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
          flashCopy();
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
