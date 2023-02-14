// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.
//

import { XY } from "./core";

export const posititonSoVisible = (target: HTMLElement, xy: XY): [XY, boolean] => {
  const { width, height } = target.getBoundingClientRect();
  const { innerWidth, innerHeight } = window;
  let changed = false;
  if (xy.x + width > innerWidth) {
    xy.x -= width;
    changed = true;
  }
  if (xy.y + height > innerHeight) {
    xy.y -= height;
    changed = true;
  }
  return [xy, changed];
}
