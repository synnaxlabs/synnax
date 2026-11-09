// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/log/Log.css";

import { clipboard } from "@synnaxlabs/lyra/clipboard";
import { context } from "@synnaxlabs/lyra/context";
import { type Flex } from "@synnaxlabs/lyra/flex";
import { Haul } from "@synnaxlabs/lyra/haul";
import { useCombinedRefs } from "@synnaxlabs/lyra/hooks";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Menu } from "@synnaxlabs/lyra/menu";
import { Status } from "@synnaxlabs/lyra/status";
import { Triggers } from "@synnaxlabs/lyra/triggers";
import { bounds, box, strings, TimeRange, type TimeStamp } from "@synnaxlabs/x";
import { type ReactElement, type ReactNode, useEffect, useRef } from "react";

import { CSS } from "@/css";
import { log } from "@/log/aether";
import { use, type UseProps } from "@/log/use";
import { canDropHaulItem, filterHaulItems } from "@/ranger/types";
import { Canvas } from "@/vis/canvas";

export const PAUSE_TRIGGER: Triggers.Trigger = ["H"];

type Mode = "selectAll" | "clearSelection" | "togglePause" | "default";

const TRIGGER_CONFIG: Triggers.ModeConfig<Mode> = {
  defaultMode: "default",
  modes: {
    selectAll: [["Control", "A"]],
    clearSelection: [Triggers.ESCAPE],
    togglePause: [PAUSE_TRIGGER],
    default: [],
  },
};

const [SeekContext, useSeekContext] = context.create<(time: TimeStamp) => void>({
  displayName: "Log.SeekContext",
  providerName: "Log.Base",
});

/** Returns a function that pauses the enclosing log at a time. */
export const useSeek = (): ((time: TimeStamp) => void) => useSeekContext("Log.useSeek");

// The width from the right edge of a log where a press grabs the scrollbar.
const SCROLLBAR_HIT_WIDTH = 12;

// Returns where the thumb sits on its track, from 0 to 1, when the pointer holding it
// grab pixels below its top is y pixels below the top of the region.
const progressAt = (
  y: number,
  grab: number,
  thumb: log.Thumb,
  regionHeight: number,
): number => {
  const travel = regionHeight - thumb.height;
  if (travel <= 0) return 0;
  return bounds.clamp({ lower: 0, upper: 1 }, (y - grab) / travel);
};

const FLATTENED_TRIGGERS = Triggers.flattenConfig(TRIGGER_CONFIG);

const toText = (lines: log.Line[]): string => lines.map((l) => l.text).join("\n");

const toTextBlob = (lines: log.Line[]): Blob =>
  new Blob([toText(lines)], { type: "text/plain" });

const toHTMLBlob = (lines: log.Line[]): Blob => {
  const body = lines.map((l) => {
    const escaped = strings.escapeHTML(l.text);
    if (l.color.length === 0) return escaped;
    // Preserve color and font when pasting into rich text editors.
    return `<span style="color: ${strings.escapeHTML(l.color)}">${escaped}</span>`;
  });
  const html = `<pre style="font-family: monospace">${body.join("\n")}</pre>`;
  return new Blob([html], { type: "text/html" });
};

const EMPTY_CONTENT = (
  <Status.Summary center level="h3" variant="disabled" hideIcon>
    No log entries
  </Status.Summary>
);

export interface BaseProps extends UseProps, Omit<Flex.BoxProps, "color"> {
  emptyContent?: ReactElement;
  extraContextMenuItems?: ReactNode;
  /** When set, the context menu offers undo and redo for the host document. */
  undoRedo?: Menu.UndoRedoItemsProps;
  enableTriggers?: Triggers.Condition;
  /** When set, a range dropped on the log pauses it at the range start, then calls
   * this with the range. */
  onDropRange?: (range: TimeRange) => void;
}

export const Base = ({
  aetherKey,
  font,
  className,
  visible,
  channelNamesHidden,
  timestampsHidden,
  timestampPrecision,
  timeZone,
  channels,
  emptyContent = EMPTY_CONTENT,
  color,
  telem,
  extraContextMenuItems,
  undoRedo,
  enableTriggers,
  hold,
  onHold,
  position,
  onLeave,
  range,
  onDropRange,
  children,
  ...rest
}: BaseProps): ReactElement | null => {
  const { state, setState, methods } = use({
    aetherKey,
    font,
    visible,
    channelNamesHidden,
    timestampsHidden,
    timestampPrecision,
    timeZone,
    channels,
    color,
    telem,
    hold,
    onHold,
    position,
    onLeave,
    range,
  });

  const {
    scrolling,
    empty,
    loading,
    blank,
    selectionStart,
    selectionEnd,
    region,
    visibleStart,
    computedLineHeight,
    rowEntries,
  } = state;

  const setHold = (hold: boolean) => {
    setState((s) => ({ ...s, scrolling: hold }));
    onHold?.(hold);
  };

  const seek = (time: TimeStamp) => {
    if (!scrolling) setHold(true);
    methods.seek(time.valueOf().toString());
  };

  const dropProps = Haul.useDrop({
    type: "log",
    canDrop: (hauled) => onDropRange != null && canDropHaulItem(hauled),
    onDrop: ({ items }) => {
      const [dropped] = filterHaulItems(items);
      if (dropped == null || onDropRange == null) return [];
      const timeRange = new TimeRange(dropped.data.timeRange);
      seek(timeRange.start);
      onDropRange(timeRange);
      return [dropped];
    },
  });
  const dragging = Haul.useDraggingState();

  const resizeRef = Canvas.useRegion((b) => setState((s) => ({ ...s, region: b })));
  const containerRef = useRef<HTMLDivElement>(null);
  const combinedRef = useCombinedRefs(resizeRef, containerRef);

  const draggingRef = useRef(false);

  const mouseYToEntryIndex = (clientY: number): number => {
    if (computedLineHeight <= 0) return 0;
    const localY = clientY - box.top(region);
    const lineIndex = Math.floor((localY - 6) / computedLineHeight);
    if (rowEntries.length === 0) return Math.max(0, visibleStart + lineIndex);
    return rowEntries[Math.min(Math.max(0, lineIndex), rowEntries.length - 1)];
  };

  const stopThumbDragRef = useRef<(() => void) | null>(null);
  useEffect(() => () => stopThumbDragRef.current?.(), []);

  // A press on the thumb drags it. A press on the track moves the thumb's middle there.
  // Either seeks on release unless the thumb did not move. A lost focus cancels.
  const dragThumb = (e: React.MouseEvent, thumb: log.Thumb): void => {
    const y = (clientY: number): number => clientY - box.top(region);
    const pressed = y(e.clientY);
    const onThumb = pressed >= thumb.top && pressed <= thumb.top + thumb.height;
    const grab = onThumb ? pressed - thumb.top : thumb.height / 2;
    const regionHeight = box.height(region);
    const pressedAt = progressAt(pressed, grab, thumb, regionHeight);
    let progress = pressedAt;
    let moved = !onThumb;
    const drag = (clientY: number): void => {
      const next = progressAt(y(clientY), grab, thumb, regionHeight);
      if (next !== pressedAt) moved = true;
      progress = next;
      setState((s) => ({ ...s, dragProgress: next }));
    };
    const stop = (): void => {
      window.removeEventListener("mousemove", handleMove);
      window.removeEventListener("mouseup", handleUp);
      window.removeEventListener("blur", handleCancel);
      stopThumbDragRef.current = null;
    };
    const end = (seek: boolean): void => {
      stop();
      setState((s) => ({ ...s, dragProgress: -1 }));
      if (seek && moved) methods.seekProgress(progress);
    };
    const handleUp = (): void => end(true);
    const handleCancel = (): void => end(false);
    // A release outside the window arrives only as a move with no button down.
    const handleMove = (move: MouseEvent): void => {
      if (move.buttons === 0) return handleUp();
      drag(move.clientY);
    };
    stopThumbDragRef.current?.();
    drag(e.clientY);
    window.addEventListener("mousemove", handleMove);
    window.addEventListener("mouseup", handleUp);
    window.addEventListener("blur", handleCancel);
    stopThumbDragRef.current = stop;
  };

  const handleMouseDown = (e: React.MouseEvent) => {
    if (e.button !== 0) return;
    if (state.thumb != null && e.clientX >= box.right(region) - SCROLLBAR_HIT_WIDTH)
      return dragThumb(e, state.thumb);
    const idx = mouseYToEntryIndex(e.clientY);
    draggingRef.current = true;
    if (e.shiftKey) setState((s) => ({ ...s, selectionEnd: idx }));
    else
      setState((s) => {
        if (s.selectionStart === idx && s.selectionEnd === idx)
          return { ...s, selectionStart: -1, selectionEnd: -1 };
        return { ...s, selectionStart: idx, selectionEnd: idx };
      });
  };

  const handleMouseMove = (e: React.MouseEvent) => {
    if (!draggingRef.current) return;
    const idx = mouseYToEntryIndex(e.clientY);
    setState((s) => ({ ...s, selectionEnd: idx }));
  };

  const handleMouseUp = () => {
    draggingRef.current = false;
  };

  const addStatus = Status.useAdder();
  const handleError = Status.useErrorHandler();

  // The worker formats the lines only now, so a large selection costs nothing until a
  // copy. The write must start inside the user gesture, so the item holds promises. An
  // insecure context has no ClipboardItem, so it copies plain text once the lines
  // arrive, and a browser refuses that copy if the gesture has expired.
  const copyToClipboard = () => {
    const lines = methods.selectedLines();
    handleError(async () => {
      if (typeof ClipboardItem === "undefined")
        await clipboard.writeText(toText(await lines));
      else
        await navigator.clipboard.write([
          new ClipboardItem({
            "text/html": lines.then(toHTMLBlob),
            "text/plain": lines.then(toTextBlob),
          }),
        ]);
      const { length } = await lines;
      addStatus({
        variant: "success",
        message: `Copied ${length} ${length === 1 ? "line" : "lines"} to clipboard`,
      });
    }, "Failed to copy to clipboard");
  };

  Triggers.use({
    triggers: FLATTENED_TRIGGERS,
    enabled: enableTriggers,
    callback: ({ triggers, stage }: Triggers.UseEvent) => {
      if (stage !== "start") return;
      const mode = Triggers.determineMode(TRIGGER_CONFIG, triggers);
      if (mode === "selectAll")
        setState((s) => ({
          ...s,
          selectionStart: 0,
          selectionEnd: log.SELECT_ALL_END,
        }));
      else if (mode === "clearSelection")
        setState((s) => ({ ...s, selectionStart: -1, selectionEnd: -1 }));
      else if (mode === "togglePause") setHold(!scrolling);
    },
  });

  const { className: menuClassName, ...menuProps } = Menu.useContextMenu();
  const hasSelection =
    selectionStart >= 0 && selectionEnd >= 0 && rowEntries.length > 0;

  const handleMenuSelect = (key: string) => {
    if (key === "copy") copyToClipboard();
  };

  const menuContent = () => (
    <Menu.Menu level="small" onChange={handleMenuSelect}>
      {undoRedo != null && (
        <>
          <Menu.UndoRedoItems {...undoRedo} />
          <Menu.Divider />
        </>
      )}
      <Menu.Item
        itemKey="copy"
        trigger={Triggers.COPY}
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
  );

  return (
    <Menu.ContextMenu className={menuClassName} menu={menuContent} {...menuProps}>
      <div
        ref={combinedRef}
        tabIndex={-1}
        className={CSS.cls(
          CSS.B("log"),
          onDropRange != null && CSS.dropRegion(canDropHaulItem(dragging)),
          className,
        )}
        {...dropProps}
        onWheel={(e) => {
          if (e.deltaY < 0 && !scrolling) setHold(true);
          setState((s) => ({ ...s, wheelPos: s.wheelPos - e.deltaY }));
        }}
        onMouseDown={handleMouseDown}
        onMouseMove={handleMouseMove}
        onMouseUp={handleMouseUp}
        onCopy={(e) => {
          if (!hasSelection) return;
          e.preventDefault();
          copyToClipboard();
        }}
        onContextMenu={menuProps.open}
        {...rest}
      >
        {blank && (
          <Status.Loading
            aria-label="Loading entries"
            className={CSS.BE("log", "blank")}
          >
            <Status.Orbital />
          </Status.Loading>
        )}
        {loading && !blank && (
          <Icon.Loading aria-label="Loading" className={CSS.BE("log", "loading")} />
        )}
        <SeekContext value={seek}>{empty ? emptyContent : children}</SeekContext>
      </div>
    </Menu.ContextMenu>
  );
};
