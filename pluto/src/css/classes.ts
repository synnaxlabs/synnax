// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Direction, Location } from "@synnaxlabs/x";

export const visibleCls = (visible?: boolean): string =>
  visible === true ? "pluto--visible" : "pluto--hidden";

export const expandedCls = (expanded?: boolean): string =>
  expanded === true ? "pluto--expanded" : "pluto--collapsed";

export const directionCls = (direction: Direction): string => `pluto--${direction}`;

export const locationCls = (location: Location): string => `pluto--${location}`;
