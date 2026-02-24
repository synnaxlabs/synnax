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
import { type ReactElement, useCallback, useEffect } from "react";
import { type z } from "zod";

import { Aether } from "@/aether";
import { Button } from "@/button";
import { CSS } from "@/css";
import { type Flex } from "@/flex";
import { Icon } from "@/icon";
import { log } from "@/log/aether";
import { useMemoDeepEqual } from "@/memo";
import { Status } from "@/status/base";
import { Canvas } from "@/vis/canvas";

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
        | "lineCount"
      >,
      "visible"
    >,
    Omit<Flex.BoxProps, "color">,
    Aether.ComponentProps {
  emptyContent?: ReactElement;
}

const getLineIndex = (node: Node | null): number | null => {
  while (node) {
    if (node instanceof HTMLElement && node.dataset.line != null)
      return parseInt(node.dataset.line);
    node = node.parentNode;
  }
  return null;
};

export const Log = ({
  aetherKey,
  font,
  className,
  visible = true,
  emptyContent = (
    <Status.Summary center level="h3" variant="disabled" hideIcon>
      Empty Log
    </Status.Summary>
  ),
  color,
  telem,
  indexTelem,
  showIndex,
  ...rest
}: LogProps): ReactElement | null => {
  const memoProps = useMemoDeepEqual({
    font,
    color,
    telem,
    visible,
    indexTelem,
    showIndex,
  });
  const [, { scrolling, empty, lineCount }, setState, methods] = Aether.use({
    type: log.Log.TYPE,
    schema: log.logState,
    methods: log.logMethodsZ,
    initialState: {
      empty: true,
      region: box.ZERO,
      scrolling: false,
      wheelPos: 0,
      lineCount: 0,
      ...memoProps,
    },
  });

  useEffect(() => {
    setState((s) => ({ ...s, ...memoProps }));
  }, [memoProps, setState]);

  const resizeRef = Canvas.useRegion(
    useCallback((b) => setState((s) => ({ ...s, region: b })), [setState]),
  );

  const handleCopy = useCallback(
    async (e: React.ClipboardEvent) => {
      e.preventDefault();
      const sel = window.getSelection();
      if (sel == null || sel.isCollapsed) return;
      const startLine = getLineIndex(sel.anchorNode);
      const endLine = getLineIndex(sel.focusNode);
      if (startLine == null || endLine == null) return;
      const start = Math.min(startLine, endLine);
      const end = Math.max(startLine, endLine) + 1;
      const text = await methods.copyText(start, end);
      await navigator.clipboard.writeText(text);
    },
    [methods],
  );

  return (
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
      {...rest}
    >
      {empty ? (
        emptyContent
      ) : (
        <>
          <div
            className={CSS.BE("log", "text-overlay")}
            onCopy={(e) => void handleCopy(e)}
          >
            {Array.from({ length: lineCount }, (_, i) => (
              <div key={i} data-line={i} className={CSS.BE("log", "line")}>
                {"\u00A0"}
              </div>
            ))}
          </div>
          <Button.Button
            className={CSS(CSS.BE("log", "live"), scrolling && CSS.M("active"))}
            variant="outlined"
            onClick={() => setState((s) => ({ ...s, scrolling: !s.scrolling }))}
            tooltip={scrolling ? "Resume Scrolling" : "Pause Scrolling"}
            tooltipLocation={location.BOTTOM_LEFT}
          >
            <Icon.Dynamic />
          </Button.Button>
          <Button.Copy
            className={CSS.BE("log", "copy")}
            text={() => methods.copyAllText()}
            tooltip="Copy all"
            tooltipLocation={location.BOTTOM_LEFT}
            variant="outlined"
          />
        </>
      )}
    </div>
  );
};
