// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ontology } from "@synnaxlabs/client";
import { Icon, Menu as PMenu, type Tree } from "@synnaxlabs/pluto";
import { type ReactElement } from "react";

import { canGroupSelection } from "@/group/canGroupSelection";

export interface MenuItemProps {
  resourceIDs: ontology.ID[];
  shape: Tree.Shape;
  showBottomDivider?: boolean;
}

export const MenuItem = ({
  resourceIDs,
  shape,
  showBottomDivider = false,
}: MenuItemProps): ReactElement | null =>
  canGroupSelection(resourceIDs, shape) ? (
    <>
      <PMenu.Item itemKey="group" startIcon={<Icon.Group />}>
        Group
      </PMenu.Item>
      {showBottomDivider && <PMenu.Divider />}
    </>
  ) : null;
