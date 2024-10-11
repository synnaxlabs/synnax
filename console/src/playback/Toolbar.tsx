import { Icon } from "@synnaxlabs/media";
import { Align, Button, Text, Timeline, TimeSpan, TimeStamp } from "@synnaxlabs/pluto";
import { useRef, useState } from "react";
import { useDispatch } from "react-redux";

import { ToolbarHeader, ToolbarTitle } from "@/components";
import { CSS } from "@/css";
import { Layout } from "@/layout";
import { useSelectCursor } from "@/playback/selector";
import { setCursor } from "@/playback/slice";
import { Range } from "@/range";

export const Content = () => {
  const cursor = useSelectCursor();
  const rng = Range.useSelect();
  const d = useDispatch();
  const onCursorChange = (cursor: number) => {
    d(setCursor({ cursor }));
  };
  const [playing, setPlaying] = useState(false);
  const playingIntervalRef = useRef<number | null>(null);
  const handlePlay = () => {
    setPlaying(!playing);
    let currPlay = cursor;
    if (playing) {
      clearInterval(playingIntervalRef.current);
      playingIntervalRef.current = null;
      return;
    }
    playingIntervalRef.current = setInterval(() => {
      currPlay += TimeSpan.milliseconds(50).nanoseconds;
      console.log(currPlay);
      d(setCursor({ cursor: currPlay }));
    }, 50);
  };

  const delta = new TimeStamp(cursor).span(new TimeStamp(rng?.timeRange.start));
  return (
    <Align.Space className={CSS.B("playback-toolbar")} empty>
      <ToolbarHeader>
        <ToolbarTitle icon={<Icon.Playback />}>Playback</ToolbarTitle>
        <Align.Space
          direction="x"
          align="center"
          style={{
            position: "absolute",
            left: "50%",
            transform: "translateX(-50%)",
          }}
        >
          <Text.DateTime
            level="p"
            shade={8}
            format="preciseDate"
            style={{ width: "130px" }}
          >
            {new TimeStamp(cursor).fString()}
          </Text.DateTime>
          <Text.Text level="p" shade={8} style={{ width: "180px" }}>
            T + {delta.toString()}
          </Text.Text>
          <Button.Icon
            variant="text"
            size="large"
            onClick={handlePlay}
            triggers={[["Control", "L"]]}
          >
            {playing ? <Icon.Pause /> : <Icon.Play />}
          </Button.Icon>
        </Align.Space>
      </ToolbarHeader>
      <Timeline.Timeline>
        <Timeline.Cursor position={cursor} onPositionChange={onCursorChange} />
        <Timeline.Track>
          {rng != null && rng.variant === "static" && (
            <Timeline.Bar
              label={rng.name}
              timeRange={rng.timeRange}
              color={"#FF0000"}
            />
          )}
        </Timeline.Track>
      </Timeline.Timeline>
    </Align.Space>
  );
};

export const Toolbar: Layout.NavDrawerItem = {
  key: "playback",
  icon: <Icon.Playback />,
  content: <Content />,
  tooltip: "Playback",
  initialSize: 150,
  minSize: 100,
  maxSize: 300,
};
