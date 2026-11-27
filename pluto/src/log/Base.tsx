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
import type { Flex } from "@synnaxlabs/lyra/flex";
import { useCombinedRefs, useSyncedRef } from "@synnaxlabs/lyra/hooks";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Menu } from "@synnaxlabs/lyra/menu";
import { Status } from "@synnaxlabs/lyra/status";
import { Text } from "@synnaxlabs/lyra/text";
import { Triggers } from "@synnaxlabs/lyra/triggers";
import { box, strings, TimeStamp, xy } from "@synnaxlabs/x";
import { type ReactElement, type ReactNode, useEffect, useRef } from "react";

import { CSS } from "@/css";
import { log } from "@/log/aether";
import { use, type UseProps } from "@/log/use";
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

const FLATTENED_TRIGGERS = Triggers.flattenConfig(TRIGGER_CONFIG);

// Pixels the log scrolls per frame for each pixel a selection drag is past its edge.
const EDGE_SCROLL_RATE = 0.2;
// A character past the end of any line.
const LINE_END = Number.MAX_SAFE_INTEGER;

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

const PARTIAL_DESCRIPTION =
  "The copy holds only the lines the log keeps in memory, not every stored value.";

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
  hiddenChannels,
  search,
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
    hiddenChannels,
    search,
    color,
    telem,
    hold,
    onHold,
    position,
    onLeave,
  });

  const {
    scrolling,
    empty,
    loading,
    blank,
    selectionStart,
    selectionEnd,
    rowEntries,
    searchedTo,
  } = state;

  const setHold = (hold: boolean) => {
    setState((s) => ({ ...s, scrolling: hold }));
    onHold?.(hold);
  };

  const resizeRef = Canvas.useRegion((b) => setState((s) => ({ ...s, region: b })));
  const containerRef = useRef<HTMLDivElement>(null);
  const combinedRef = useCombinedRefs(resizeRef, containerRef);

  const stateRef = useSyncedRef(state);
  const pointerRef = useRef<xy.XY>(xy.ZERO);
  const frameRef = useRef<number | null>(null);
  const detachDragRef = useRef<(() => void) | null>(null);
  useEffect(() => () => detachDragRef.current?.(), []);
  // The character the pointer went down on, while it drags.
  const downRef = useRef<log.Point | null>(null);
  // Counts presses and releases, so a hit that resolves after its release starts no
  // drag.
  const gestureRef = useRef(0);
  // Counts presses and moves, so only the hit of the last move ends a drag.
  const moveRef = useRef(0);

  const addStatus = Status.useAdder();
  const handleError = Status.useErrorHandler();

  // The row of the view at the pointer, counted from the top.
  const rowAt = (clientY: number): number => {
    const { computedLineHeight, region } = stateRef.current;
    if (computedLineHeight <= 0) return 0;
    return Math.max(
      0,
      Math.floor((clientY - box.top(region) - 6) / computedLineHeight),
    );
  };

  // How far the pointer is above the view (negative) or below it (positive).
  const pastEdge = (clientY: number): number => {
    const { region } = stateRef.current;
    return (
      Math.min(0, clientY - box.top(region)) + Math.max(0, clientY - box.bottom(region))
    );
  };

  // Past an edge, a drag ends at the start of the row above the view or past the end of
  // the row below it. The row follows the view as it scrolls.
  const edgePoint = (clientY: number, past: number): log.Point => ({
    entry: log.selectRow(rowAt(clientY)),
    char: past < 0 ? 0 : LINE_END,
  });

  // Moves the end of the selection that a drag from down makes to point.
  const extend = (down: log.Point, point: log.Point) =>
    setState((s) => {
      // The press selected its line, and the worker keeps that start on its entry as
      // entries shift, so the drag reads the start from the state. A start the worker
      // cleared ends the drag.
      const start = s.selectionStart;
      if (start < 0) {
        if (downRef.current === down) downRef.current = null;
        return s;
      }
      const startChar = s.selectionStartChar >= 0 ? s.selectionStartChar : down.char;
      // A drag back to its start returns to the line the press selected.
      if (point.entry === start && point.char === startChar)
        return {
          ...s,
          selectionEnd: start,
          selectionStartChar: -1,
          selectionEndChar: -1,
        };
      return {
        ...s,
        selectionStart: start,
        selectionStartChar: startChar,
        selectionEnd: point.entry,
        selectionEndChar: point.char,
      };
    });

  const scrollPastEdge = () => {
    frameRef.current = null;
    const down = downRef.current;
    const { y } = pointerRef.current;
    const past = pastEdge(y);
    if (past === 0 || down == null) return;
    if (past < 0 && !stateRef.current.scrolling) setHold(true);
    moveRef.current++;
    setState((s) => ({ ...s, wheelPos: s.wheelPos - past * EDGE_SCROLL_RATE }));
    extend(down, edgePoint(y, past));
    frameRef.current = requestAnimationFrame(scrollPastEdge);
  };

  // A drag selects from the character it started on to the one under the pointer.
  const startDrag = (pointer: xy.XY) => {
    pointerRef.current = pointer;
    const handleMove = (e: MouseEvent) => {
      // A release outside the window sends no mouseup.
      if ((e.buttons & 1) === 0) {
        stop();
        return;
      }
      const { clientX: x, clientY: y } = e;
      pointerRef.current = { x, y };
      const down = downRef.current;
      if (down == null) return;
      const move = ++moveRef.current;
      const past = pastEdge(y);
      if (past !== 0) {
        extend(down, edgePoint(y, past));
        if (frameRef.current == null)
          frameRef.current = requestAnimationFrame(scrollPastEdge);
        return;
      }
      handleError(async () => {
        const point = await methods.hit({ x, y });
        // A move whose hit resolves after the release still ends the drag.
        if (move === moveRef.current) extend(down, point);
      }, "Failed to select log lines");
    };
    const detach = () => {
      window.removeEventListener("mousemove", handleMove);
      window.removeEventListener("mouseup", stop);
      window.removeEventListener("blur", stop);
      if (frameRef.current != null) cancelAnimationFrame(frameRef.current);
      frameRef.current = null;
      detachDragRef.current = null;
    };
    const stop = () => {
      detach();
      downRef.current = null;
      gestureRef.current++;
      setState((s) => ({ ...s, selecting: false }));
    };
    detachDragRef.current?.();
    detachDragRef.current = detach;
    setState((s) => ({ ...s, selecting: true }));
    window.addEventListener("mousemove", handleMove);
    window.addEventListener("mouseup", stop);
    window.addEventListener("blur", stop);
  };

  // A click selects its whole line, or clears it when it alone is selected. A
  // shift-click extends the selection by whole lines. Neither a shift-click nor a click
  // that clears starts a drag.
  const handleMouseDown = (e: React.MouseEvent) => {
    if (e.button !== 0) return;
    const { clientX: x, clientY: y, shiftKey } = e;
    downRef.current = null;
    moveRef.current++;
    const gesture = ++gestureRef.current;
    if (!shiftKey) startDrag({ x, y });
    handleError(async () => {
      const point = await methods.hit({ x, y });
      const press = !shiftKey && gesture === gestureRef.current ? point : null;
      if (press != null) downRef.current = press;
      const { entry } = point;
      setState((s) => {
        const whole = { selectionStartChar: -1, selectionEndChar: -1 };
        if (shiftKey) return { ...s, ...whole, selectionEnd: entry };
        const alone =
          s.selectionStart === entry &&
          s.selectionEnd === entry &&
          s.selectionStartChar < 0;
        if (alone && press != null && downRef.current === press) downRef.current = null;
        if (alone) return { ...s, ...whole, selectionStart: -1, selectionEnd: -1 };
        return { ...s, ...whole, selectionStart: entry, selectionEnd: entry };
      });
    }, "Failed to select log lines");
  };

  // The worker formats the lines only now, so a large selection costs nothing until a
  // copy. The write must start inside the user gesture, so the item holds promises. An
  // insecure context has no ClipboardItem, so it copies plain text once the lines
  // arrive, and a browser refuses that copy if the gesture has expired.
  const copyToClipboard = () => {
    const selection = methods.selection();
    const lines = selection.then((s) => s.lines);
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
      const message = `Copied ${strings.count(length, "line")} to clipboard`;
      if ((await selection).partial)
        addStatus({ variant: "warning", message, description: PARTIAL_DESCRIPTION });
      else addStatus({ variant: "success", message });
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
          selectionStartChar: -1,
          selectionEndChar: -1,
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
    <Menu.Menu onChange={handleMenuSelect}>
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
        role="log"
        tabIndex={-1}
        className={CSS.cls(CSS.B("log"), className)}
        onWheel={(e) => {
          if (e.deltaY < 0 && !scrolling) setHold(true);
          setState((s) => ({ ...s, wheelPos: s.wheelPos - e.deltaY }));
        }}
        onMouseDown={handleMouseDown}
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
        {searchedTo !== "" && (
          <Text.Text level="small" color={9} className={CSS.BE("log", "searched")}>
            Searched back to{" "}
            {new TimeStamp(BigInt(searchedTo)).toString("dateTime", timeZone)}
          </Text.Text>
        )}
        {empty && emptyContent}
        {children}
      </div>
    </Menu.ContextMenu>
  );
};
