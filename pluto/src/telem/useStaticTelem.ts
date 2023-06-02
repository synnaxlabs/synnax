import { useTelemSourceControl } from "./Context";
import { DynamicTelemProps, StaticTelemProps } from "./staticTelem";

import { XYTelemSourceMeta } from "@/core/vis/telem";

export const useStaticTelem = (props: StaticTelemProps): XYTelemSourceMeta => {
  const key = useTelemSourceControl("static", props, [
    ...props.x.map((x) => x.buffer),
    ...props.y.map((y) => y.buffer),
  ]);
  return {
    type: "xy",
    key,
  };
};

export const useDynamicTelem = (props: DynamicTelemProps): XYTelemSourceMeta => {
  const key = useTelemSourceControl("dynamic", props, [
    ...props.x.map((x) => x.buffer),
    ...props.y.map((y) => y.buffer),
  ]);
  return {
    type: "xy",
    key,
  };
};
