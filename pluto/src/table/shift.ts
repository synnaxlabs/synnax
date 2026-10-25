// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { context } from "@synnaxlabs/lyra/context";
import { xy } from "@synnaxlabs/x";

/** Shift carries how far a cell slides while a move previews where it lands. */
export const [Shift, useShift] = context.create<xy.XY>({
  defaultValue: xy.ZERO,
  displayName: "Table.Shift",
});

/** Returns the CSS transform that slides an element by shift. */
export const transform = (shift: xy.XY): string | undefined =>
  xy.equals(shift, xy.ZERO) ? undefined : `translate(${shift.x}px, ${shift.y}px)`;
