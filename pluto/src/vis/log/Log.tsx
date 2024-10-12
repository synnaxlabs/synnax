// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/vis/log/log.css";

import { box } from "@synnaxlabs/x";
import { ReactElement, useCallback, useEffect, useRef } from "react";
import { z } from "zod";

import { Aether } from "@/aether";
import { Align } from "@/align";
import { Button } from "@/button";
import { CSS } from "@/css";
import { useCombinedRefs, useSyncedRef } from "@/hooks";
import { useMemoDeepEqualProps } from "@/memo";
import { Triggers } from "@/triggers";
import { Canvas } from "@/vis/canvas";
import { log } from "@/vis/log/aether";

export interface LogProps
  extends Omit<
      z.input<typeof log.logState>,
      "region" | "totalHeight" | "scrollPosition"
    >,
    Omit<Align.SpaceProps, "color"> {}

export const Log = Aether.wrap<LogProps>(
  "Log",
  ({ aetherKey, font, className, color, telem, ...props }): ReactElement | null => {
    const memoProps = useMemoDeepEqualProps({ font, color, telem });
    const elRef = useRef<HTMLDivElement | null>(null);
    const [, { scrollPosition, totalHeight }, setState] = Aether.use({
      aetherKey,
      type: log.Log.TYPE,
      schema: log.logState,
      initialState: {
        region: box.ZERO,
        scrollPosition: null,
        totalHeight: 0,
        ...memoProps,
      },
    });

    const scrollPosRef = useSyncedRef(scrollPosition);
    const snapRef = useRef<number | null>(null);
    useEffect(() => {
      if (elRef.current == null || snapRef.current != null) return;
      elRef.current.scrollTop = elRef.current.scrollHeight ?? 0;
    }, [totalHeight]);

    useEffect(() => {
      setState((s) => ({ ...s, ...memoProps }));
    }, [memoProps, setState]);

    const resizeRef = Canvas.useRegion(
      useCallback(
        (b) => {
          if (snapRef.current == null && elRef.current != null)
            elRef.current.scrollTop = elRef.current.scrollHeight;
          setState((s) => ({ ...s, region: b }));
        },
        [setState],
      ),
    );

    const logRef = useRef<HTMLDivElement | null>(null);

    Triggers.use({
      triggers: [["Control", "MouseLeft"]],
      region: logRef,
      regionMustBeElement: false,
      callback: useCallback(({ stage }: Triggers.UseEvent) => {
        if (stage !== "start") return;
        elRef.current?.scrollTo({ top: elRef.current?.scrollHeight });
        setState((s) => ({ ...s, scrollPosition: null }));
      }, []),
    });

    const combinedRef = useCombinedRefs(elRef, resizeRef);
    return (
      <div ref={logRef} className={CSS(CSS.B("log"), className)} {...props}>
        {scrollPosition != null && (
          <Button.Button
            style={{ position: "absolute", top: 0, right: 0 }}
            variant="text"
            onClick={() => {
              elRef.current?.scrollTo({ top: elRef.current?.scrollHeight });
              setState((s) => ({ ...s, scrollPosition: null }));
            }}
          >
            Back to live
          </Button.Button>
        )}
        <div
          className={CSS.BE("log", "scroll")}
          ref={combinedRef}
          onScroll={(e) => {
            const el = e.target as HTMLDivElement;
            const elScrollPos = el.scrollTop + el.clientHeight;
            if (elScrollPos > el.scrollHeight - 100) {
              snapRef.current = null;
              if (scrollPosRef.current != null)
                setState((s) => ({ ...s, scrollPosition: null }));
              return;
            }
            if (snapRef.current == null) snapRef.current = el.scrollHeight;
            setState((s) => ({
              ...s,
              scrollPosition: elScrollPos - (snapRef.current ?? el.scrollHeight),
            }));
          }}
        >
          <div style={{ height: totalHeight * 1.02 }} />
        </div>
      </div>
    );
  },
);
