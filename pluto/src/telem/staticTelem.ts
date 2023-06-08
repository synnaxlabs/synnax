import { Bounds, GLBufferControl, LazyArray, NativeTypedArray } from "@synnaxlabs/x";

import { DynamicXYTelemSource, TelemSourceMeta } from "@/core/vis/telem/TelemSource";

export class StaticTelemFactory {
  new(
    key: string,
    type: string,
    props: StaticTelemProps | DynamicTelemProps
  ): TelemSourceMeta {
    if (type === "dynamic") return new DynamicTelem(key, props as DynamicTelemProps);
    return new StaticTelem(key, props);
  }
}
