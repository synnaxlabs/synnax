// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { xy } from "@synnaxlabs/x";

export const calcPath = (coords: xy.XY[]): string => {
  let path = "";
  const close = false;
  const radius = 6;
  const length = coords.length + (close ? 1 : -1);
  for (let i = 0; i < length; i++) {
    const a = coords[i % coords.length];
    const b = coords[(i + 1) % coords.length];
    const t = Math.min(radius / Math.hypot(b.x - a.x, b.y - a.y), 0.5);
    if (i > 0)
      path += `Q${a.x},${a.y} ${a.x * (1 - t) + b.x * t},${a.y * (1 - t) + b.y * t}`;
    if (!close && i === 0) path += `M${a.x},${a.y}`;
    else if (i === 0) path += `M${a.x * (1 - t) + b.x * t},${a.y * (1 - t) + b.y * t}`;
    if (!close && i === length - 1) path += `L${b.x},${b.y}`;
    else if (i < length - 1)
      path += `L${a.x * t + b.x * (1 - t)},${a.y * t + b.y * (1 - t)}`;
  }
  if (close) path += "Z";
  return path;
};
