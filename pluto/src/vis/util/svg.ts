// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Location, XY } from "@synnaxlabs/x";

export const fTranslate = (v: XY): string => `translate(${v.x}, ${v.y})`;

export const fRotate = (v: number): string => `rotate(${v})`;

export const locationRotations: Record<Location, number> = {
  bottom: 180,
  top: 0,
  left: -90,
  right: 90,
  center: 0,
};
