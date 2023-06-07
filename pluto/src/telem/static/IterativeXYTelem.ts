// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { LazyArray, NativeTypedArray } from "@synnaxlabs/x";

import { DynamicXYTelemSource } from "@/core/vis/telem";
import { StaticXYTelem } from "@/telem/static/StaticXYTelem";

export interface IterativeXYTelemProps {
  x: NativeTypedArray[];
  y: NativeTypedArray[];
  updateRate: number;
}

export class IterativeXYTelem extends StaticXYTelem implements DynamicXYTelemSource {
  handler: (() => void) | null;
  position: number;

  type = "xy";

  constructor(key: string, props: IterativeXYTelemProps) {
    super(key, props);
    this.handler = null;
    this.position = 0;

    setInterval(() => {
      if (this.handler != null) this.handler();
      this.position++;
    }, props.updateRate);
  }

  onChange(f: () => void): void {
    this.handler = f;
  }

  setProps(props: IterativeXYTelemProps): void {
    this._x = props.x.map((x) => new LazyArray(x));
    this._y = props.y.map((y) => new LazyArray(y));
  }
}
