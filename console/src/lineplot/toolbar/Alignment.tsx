import "@/lineplot/toolbar/Alignment.css";

import { Theming, Timeline, TimeSpan } from "@synnaxlabs/pluto";
import { useDispatch } from "react-redux";

import { CSS } from "@/css";
import { useSelectRanges } from "@/lineplot/selectors";
import { ToolbarProps } from "@/lineplot/toolbar/Toolbar";
import { Range } from "@/range";
import { StaticRange } from "@/range/slice";

export const Alignment = ({ layoutKey }: ToolbarProps): ReactElement => {
  const ranges = useSelectRanges(layoutKey);
  const theme = Theming.use();
  const newColors = theme?.colors.visualization.palettes.default ?? [];
  const d = useDispatch();

  const specs: Timeline.BarSpec[] = Object.values(ranges)
    .flat()
    .filter((r) => r.variant === "static")
    .map((r: StaticRange, i) => ({
      key: r.key,
      label: r.name,
      timeRange: r.timeRange,
      color: newColors[i % newColors.length],
      offset: r.offset,
    }));

  const handleTranslate = (key: string, offset: TimeSpan) => {
    d(Range.setOffset({ key, offset: Number(offset.valueOf()) }));
  };

  return (
    <Timeline.Timeline
      className={CSS.B("alignment")}
      onTranslate={handleTranslate}
      bars={specs}
    />
  );
};
