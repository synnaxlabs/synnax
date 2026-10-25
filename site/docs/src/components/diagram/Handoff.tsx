// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/components/diagram/Handoff.css";

import { type CSSProperties, Fragment, type ReactElement } from "react";

export interface HandoffWriter {
  name: string;
  caption?: string;
  /** The writer's authority after each event, or null while the writer is closed. */
  authority: (number | null)[];
}

export interface HandoffEvent {
  label: string;
  /** Why the writer that controls the channel after the event has control. */
  reason: string;
}

export interface HandoffProps {
  /** Describes the diagram for screen readers. */
  title: string;
  writers: HandoffWriter[];
  events: HandoffEvent[];
}

interface Bar {
  start: number;
  end: number;
  authority: number;
  controlling: boolean;
}

/**
 * Returns the index of the writer that controls the channel after each event. The
 * highest authority controls, and the writer that opened first wins a tie.
 */
const controllers = (writers: HandoffWriter[], count: number): (number | null)[] => {
  const opened = writers.map(() => -1);
  const result: (number | null)[] = [];
  for (let i = 0; i < count; i++) {
    let controller: number | null = null;
    let highest = -1;
    for (let w = 0; w < writers.length; w++) {
      const a = writers[w].authority[i];
      if (a == null) {
        opened[w] = -1;
        continue;
      }
      if (opened[w] === -1) opened[w] = i;
      if (
        controller == null ||
        a > highest ||
        (a === highest && opened[w] < opened[controller])
      ) {
        controller = w;
        highest = a;
      }
    }
    result.push(controller);
  }
  return result;
};

const bars = (authority: (number | null)[], controlling: boolean[]): Bar[] => {
  const result: Bar[] = [];
  authority.forEach((a, i) => {
    if (a == null) return;
    const last = result.at(-1);
    if (last?.end === i && last.authority === a && last.controlling === controlling[i])
      last.end = i + 1;
    else
      result.push({ start: i, end: i + 1, authority: a, controlling: controlling[i] });
  });
  return result;
};

const Swatch = ({ variant }: { variant: "controlling" | "blocked" }): ReactElement => (
  <i className={`handoff-diagram__swatch handoff-diagram__bar--${variant}`} />
);

/**
 * Draws how control of one channel passes between writers over a numbered series of
 * events. Each writer gets a track of bars labeled with its authority, and a numbered
 * list under the tracks says which writer controls the channel after each event.
 */
export const Handoff = ({ title, writers, events }: HandoffProps): ReactElement => {
  const count = events.length;
  for (const { name, authority } of writers)
    if (authority.length !== count)
      throw new Error(`${name} needs an authority for each of the ${count} events`);
  const control = controllers(writers, count);
  const style = {
    "--events": count,
    "--writers": writers.length,
    "--event-rows": Math.ceil(count / 2),
  } as CSSProperties;
  return (
    <figure className="handoff-diagram" style={style}>
      <div className="handoff-diagram__grid" role="img" aria-label={title}>
        {control.map((_, i) => (
          <span
            key={i}
            className="handoff-diagram__event"
            style={{ gridColumn: i + 2 }}
          >
            <span className="handoff-diagram__badge">{i + 1}</span>
          </span>
        ))}
        {writers.map(({ name, caption, authority }, w) => (
          <Fragment key={name}>
            <span className="handoff-diagram__writer" style={{ gridRow: w + 2 }}>
              <span className="handoff-diagram__name">{name}</span>
              {caption != null && (
                <span className="handoff-diagram__caption">{caption}</span>
              )}
            </span>
            {bars(
              authority,
              control.map((c) => c === w),
            ).map(({ start, end, authority: a, controlling }) => (
              <span
                key={start}
                className={[
                  "handoff-diagram__bar",
                  `handoff-diagram__bar--${controlling ? "controlling" : "blocked"}`,
                  end === count ? "handoff-diagram__bar--open" : "",
                ].join(" ")}
                style={{ gridRow: w + 2, gridColumn: `${start + 2} / ${end + 2}` }}
              >
                <span className="handoff-diagram__value">{a}</span>
              </span>
            ))}
          </Fragment>
        ))}
      </div>
      <figcaption className="handoff-diagram__legend">
        <span>
          <Swatch variant="controlling" />
          Controls the channel
        </span>
        <span>
          <Swatch variant="blocked" />
          Open, values dropped
        </span>
        <span>
          <span className="handoff-diagram__legend-value">255</span>
          Authority
        </span>
      </figcaption>
      <ol className="handoff-diagram__events">
        {events.map(({ label, reason }, i) => {
          const c = control[i];
          return (
            <li key={i}>
              <span className="handoff-diagram__number">{i + 1}</span>
              <span className="handoff-diagram__event-label">{label}</span>
              <span className="handoff-diagram__event-result">
                {c == null ? (
                  "No writer controls the channel."
                ) : (
                  <>
                    <strong>{writers[c].name}</strong>
                    {i > 0 && control[i - 1] === c ? " keeps" : " takes"} control.{" "}
                    {reason}
                  </>
                )}
              </span>
            </li>
          );
        })}
      </ol>
    </figure>
  );
};
