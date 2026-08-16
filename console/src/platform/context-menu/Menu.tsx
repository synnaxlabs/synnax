// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Menu as PMenu } from "@synnaxlabs/pluto";
import { type PropsWithChildren, type ReactElement } from "react";

export interface MenuProps extends PropsWithChildren {}

/**
 * Menu is the root of every console context menu. Order items in fixed bands,
 * top to bottom: primary action (what double-click would do), rename and
 * organization, feature-specific actions, Export / Copy link / Copy properties,
 * Delete, Reload Console. Emit a Menu.Divider between bands unconditionally;
 * dividers next to nothing hide themselves.
 */
export const Menu = (props: MenuProps): ReactElement => (
  <PMenu.Menu {...props} level="small" gap="small" />
);
