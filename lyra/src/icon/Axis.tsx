// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ReactElement } from "react";

import { type FC, wrapSVGIcon } from "@/icon/Icon";

const FRAME =
  "M4 3h16a1 1 0 0 1 1 1v16a1 1 0 0 1-1 1H4a1 1 0 0 1-1-1V4a1 1 0 0 1 1-1Zm.5 1.5v15h15v-15Z";

/* #__NO_SIDE_EFFECTS__ */
const createAxis = (name: string, side: ReactElement): FC =>
  wrapSVGIcon(
    (props) => (
      <svg
        viewBox="0 0 24 24"
        xmlns="http://www.w3.org/2000/svg"
        stroke="currentColor"
        fill="currentColor"
        strokeWidth={0}
        {...props}
      >
        <path fillRule="evenodd" clipRule="evenodd" opacity={0.4} d={FRAME} />
        {side}
      </svg>
    ),
    name,
  );

/** A plot frame with the side an axis sits on drawn solid. */
export const Axis = {
  Left: createAxis("axis-left", <rect x={3} y={3} width={4} height={18} rx={1} />),
  Right: createAxis("axis-right", <rect x={17} y={3} width={4} height={18} rx={1} />),
  Top: createAxis("axis-top", <rect x={3} y={3} width={18} height={4} rx={1} />),
  Bottom: createAxis("axis-bottom", <rect x={3} y={17} width={18} height={4} rx={1} />),
};
