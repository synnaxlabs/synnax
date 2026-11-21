// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/feature/range/toolbar/Card.css";

import type { ranger } from "@synnaxlabs/client";
import { useNow } from "@synnaxlabs/lyra/hooks";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Store } from "@synnaxlabs/lyra/store";
import { Tag } from "@synnaxlabs/lyra/tag";
import { Telem } from "@synnaxlabs/lyra/telem";
import { Text } from "@synnaxlabs/lyra/text";
import { Color, Flux, Ranger } from "@synnaxlabs/pluto";
import { color, TimeSpan, TimeStamp } from "@synnaxlabs/x";
import { type CSSProperties, Fragment, type ReactElement } from "react";

import { formatSpan, nextSecond } from "@/feature/range/toolbar/time";
import { CSS } from "@/platform/css";

const B = "range-card";
const CHILD_LIMIT = 6;
const OLDEST: ranger.Order = { field: "time_range_start", descending: false };

const at = (t: TimeStamp, now: TimeStamp, day: boolean): string => {
  const time = Telem.Text.formatTime(t, TimeSpan.SECOND);
  return day ? `${Telem.Text.describeDay(t, now)} ${time}` : time;
};

const colorStyle = (c?: color.Color): CSSProperties =>
  (c == null ? {} : { "--console-range-color": color.hex(c) }) as CSSProperties;

const usePath = (range: ranger.Range): string[] => {
  const links = Ranger.useResultAncestors({ keys: [range.key] }).data;
  const chain: ranger.Key[] = [];
  let k = range.parent ?? null;
  while (k != null && links != null) {
    chain.unshift(k);
    k = links[k] ?? null;
  }
  const { data } = Ranger.useResultMultiple(
    chain.length === 0 ? null : { keys: chain },
  );
  const names = new Map((data ?? []).map((r) => [r.key, r.name]));
  return chain.map((c) => names.get(c)).filter((n): n is string => n != null);
};

interface Children {
  children: ranger.Range[];
  /** Whether `children` holds every child, not only the first page. */
  complete: boolean;
}

const useChildren = (range: ranger.Range): Children => {
  const list = Flux.useList((c) => c.ranges, {
    name: "ranges",
    query: { where: { parents: [range.key] }, order: OLDEST },
    initialSize: 50,
    eager: true,
  });
  const children = Store.useItems({
    keys: list.data,
    subscribe: list.subscribe,
    getItem: list.getItem,
  });
  return { children, complete: list.complete };
};

export const Card = ({ range }: { range: ranger.Range }): ReactElement => {
  const now = useNow(nextSecond);
  const path = usePath(range);
  const { children, complete } = useChildren(range);
  const labels = Ranger.useLabels(range.key) ?? [];
  const resolve = Color.useResolve();
  const tr = range.timeRange.makeValid();
  const stage = Ranger.getStage(range.timeRange, now);
  const StageIcon = Ranger.STAGE_ICONS[stage];
  const live = stage === "in_progress";
  const openEnd = !live && tr.end.equals(TimeStamp.MAX);
  const end = live ? now : tr.end;
  const total = Math.max(1, end.nanoseconds - tr.start.nanoseconds);
  const sameDay =
    Telem.Text.describeDay(tr.start, now) === Telem.Text.describeDay(end, now);
  const position = (r: ranger.Range): CSSProperties => {
    const v = r.timeRange.makeValid();
    const e = Ranger.getStage(r.timeRange, now) === "in_progress" ? now : v.end;
    const left = (v.start.nanoseconds - tr.start.nanoseconds) / total;
    const width = (e.nanoseconds - v.start.nanoseconds) / total;
    return {
      ...colorStyle(resolve(r.color)),
      left: `${Math.max(0, left) * 100}%`,
      width: `${Math.max(0.006, Math.min(1, width)) * 100}%`,
    };
  };
  return (
    <div className={CSS.B(B)} style={colorStyle(resolve(range.color))}>
      <div className={CSS.BE(B, "head")}>
        <span className={CSS.BE(B, "mark")} />
        <span className={CSS.BE(B, "name")}>{range.name}</span>
        <span className={CSS.BE(B, "stage")}>
          <StageIcon />
          {live ? "Running" : Ranger.STAGE_NAMES[stage]}
        </span>
      </div>
      {path.length > 0 && (
        <div className={CSS.BE(B, "path")}>
          {path.map((name, i) => (
            <Fragment key={i}>
              {i > 0 && <Icon.Caret.Right />}
              <span>{name}</span>
            </Fragment>
          ))}
        </div>
      )}
      <div className={CSS.BE(B, "time")}>
        <span>
          {at(tr.start, now, true)}
          {!openEnd && (
            <>
              <Icon.Arrow.Right />
              {live ? "Now" : at(end, now, !sameDay)}
            </>
          )}
        </span>
        {!openEnd && (
          <span className={CSS.BE(B, "total")}>{formatSpan(tr.start, end)}</span>
        )}
      </div>
      {children.length > 0 && (
        <>
          {/* A strip needs both ends and every child to place them in time. */}
          {complete && !openEnd && (
            <div className={CSS.BE(B, "strip")}>
              {children.map((c) => (
                <span key={c.key} style={position(c)} />
              ))}
            </div>
          )}
          <div className={CSS.BE(B, "children")}>
            {children.slice(0, CHILD_LIMIT).map((c) => {
              const v = c.timeRange.makeValid();
              const running = Ranger.getStage(c.timeRange, now) === "in_progress";
              return (
                <div
                  key={c.key}
                  className={CSS.cls(
                    CSS.BE(B, "child"),
                    running && CSS.BEM(B, "child", "live"),
                  )}
                  style={colorStyle(resolve(c.color))}
                >
                  <span className={CSS.BE(B, "mark")} />
                  <span className={CSS.BE(B, "child-name")}>{c.name}</span>
                  <span>{formatSpan(v.start, running ? now : v.end)}</span>
                </div>
              );
            })}
            {children.length > CHILD_LIMIT && (
              <Text.Text level="small" color={9} className={CSS.BE(B, "more")}>
                {children.length - CHILD_LIMIT}
                {complete ? "" : "+"} more
              </Text.Text>
            )}
          </div>
        </>
      )}
      {labels.length > 0 && (
        <div className={CSS.BE(B, "labels")}>
          {labels.map((l) => (
            <Tag.Tag key={l.key} color={resolve(l.color)} size="tiny">
              {l.name}
            </Tag.Tag>
          ))}
        </div>
      )}
    </div>
  );
};
