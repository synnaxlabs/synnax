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
import { type Flex } from "@synnaxlabs/lyra/flex";
import { useCombinedRefs } from "@synnaxlabs/lyra/hooks";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Menu } from "@synnaxlabs/lyra/menu";
import { Status } from "@synnaxlabs/lyra/status";
import { Text } from "@synnaxlabs/lyra/text";
import { Triggers } from "@synnaxlabs/lyra/triggers";
import { strings, TimeStamp } from "@synnaxlabs/x";
import { type ReactElement, type ReactNode, useRef } from "react";

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

  // The character the pointer went down on, while it drags.
  const downRef = useRef<log.Point | null>(null);
  // Counts presses and releases, so a hit that resolves after its release starts no
  // drag.
  const gestureRef = useRef(0);

  const addStatus = Status.useAdder();
  const handleError = Status.useErrorHandler();

  // A click selects its whole line, or clears it when it alone is selected. A
  // shift-click extends the selection by whole lines. Neither a shift-click nor a click
  // that clears starts a drag.
  const handleMouseDown = (e: React.MouseEvent) => {
    if (e.button !== 0) return;
    const { clientX: x, clientY: y, shiftKey } = e;
    downRef.current = null;
    const gesture = ++gestureRef.current;
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

  // A drag selects from the character it started on to the one under the pointer.
  const handleMouseMove = (e: React.MouseEvent) => {
    if (downRef.current == null) return;
    const { clientX: x, clientY: y } = e;
    handleError(async () => {
      const point = await methods.hit({ x, y });
      const down = downRef.current;
      if (down == null) return;
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
    }, "Failed to select log lines");
  };

  const handleMouseUp = () => {
    downRef.current = null;
    gestureRef.current++;
  };

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
        role="log"
        tabIndex={-1}
        className={CSS.cls(CSS.B("log"), className)}
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
