// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Button } from "@synnaxlabs/lyra/button";
import { Flex } from "@synnaxlabs/lyra/flex";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Select } from "@synnaxlabs/lyra/select";
import { Text } from "@synnaxlabs/lyra/text";
import { type TimeRange, TimeSpan, type TimeStamp } from "@synnaxlabs/x";
import { type ReactElement, useEffect, useState } from "react";

import { CSS } from "@/css";
import { clock } from "@/video/clock";

const SPEEDS = [0.25, 0.5, 1, 2, 4, 8];

export interface ControlsProps {
  /** The range the video plays. */
  range: TimeRange;
  clock: clock.Clock;
  onChange: (clock: clock.Clock) => void;
}

// Returns the time the clock shows, updated on each animation frame while it plays.
// The clock pauses at the end of the range.
const useTime = ({ range, clock: c, onChange }: ControlsProps): TimeStamp => {
  const [wall, setWall] = useState(Date.now);
  useEffect(() => {
    if (!c.playing) return;
    let frame = 0;
    const tick = (): void => {
      setWall(Date.now());
      frame = requestAnimationFrame(tick);
    };
    frame = requestAnimationFrame(tick);
    return () => cancelAnimationFrame(frame);
  }, [c]);
  const time = clock.timeAt(c, Math.max(wall, c.wall));
  const ended = c.playing && time.afterEq(range.end);
  useEffect(() => {
    if (ended) onChange(clock.pause(clock.seek(c, range.end)));
  }, [ended]);
  return ended ? range.end : time;
};

/** Plays, pauses, scrubs, and sets the speed of a video over a range. */
export const Controls = (props: ControlsProps): ReactElement => {
  const { range, clock: c, onChange } = props;
  const time = useTime(props);
  const span = range.span.milliseconds;
  const handlePlay = (): void => {
    if (c.playing) return onChange(clock.pause(c));
    const from = time.afterEq(range.end) ? clock.seek(c, range.start) : c;
    onChange(clock.play(from));
  };
  return (
    <Flex.Box x align="center" gap="small" className={CSS.BE("video", "controls")}>
      <Button.Button
        variant="text"
        size="small"
        tooltip={c.playing ? "Pause" : "Play"}
        onClick={handlePlay}
      >
        {c.playing ? <Icon.Pause /> : <Icon.Play />}
      </Button.Button>
      <input
        type="range"
        aria-label="Time"
        className={CSS.BE("video", "scrub")}
        min={0}
        max={span}
        value={time.span(range.start).milliseconds}
        onChange={(e) =>
          onChange(
            clock.seek(
              c,
              range.start.add(TimeSpan.milliseconds(Number(e.target.value))),
            ),
          )
        }
      />
      <Text.Text level="small" color={9}>
        {time.toString("preciseTime", "local")}
      </Text.Text>
      <Select.Simple<number>
        value={c.speed}
        onChange={(speed: number) => onChange(clock.setSpeed(c, speed))}
        resourceName="speed"
      >
        {SPEEDS.map((s) => (
          <Select.Item key={s} itemKey={s}>
            {`${s}x`}
          </Select.Item>
        ))}
      </Select.Simple>
    </Flex.Box>
  );
};
