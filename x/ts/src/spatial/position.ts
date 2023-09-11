// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import * as xy from "@/spatial/xy";

export const posititonSoVisible = (target: HTMLElement, p: xy.XY): [xy.XY, boolean] => {
  const { width, height } = target.getBoundingClientRect();
  const { innerWidth, innerHeight } = window;
  let changed = false;
  let nextXY = xy.construct(p);
  if (p.x + width > innerWidth) {
    nextXY = xy.translateX(nextXY, -width);
    changed = true;
  }
  if (p.y + height > innerHeight) {
    nextXY = xy.translateY(nextXY, -height);
    changed = true;
  }
  return [nextXY, changed];
};
