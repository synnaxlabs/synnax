// Copyright 2022 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Nav } from "@synnaxlabs/pluto";

import { NAV_SIZES } from "./constants";

/**
 * NavTop is the top navigation bar for the Delta UI. Try to keep this component
 * presentational.
 */
export const NavTop = (): JSX.Element => (
  <Nav.Bar data-tauri-drag-region location="top" size={NAV_SIZES.top} />
);
