// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Icon, Menu as PMenu } from "@synnaxlabs/pluto";
import { type ReactElement } from "react";

import { canGroupSelection } from "@/group/canGroupSelection";
import { type Ontology } from "@/ontology";

export interface MenuItemProps {
  selection: Ontology.TreeContextMenuProps["selection"];
  showBottomDivider?: boolean;
}

export const MenuItem = ({
  selection,
  showBottomDivider = false,
}: MenuItemProps): ReactElement | null =>
  canGroupSelection(selection) ? (
    <>
      <PMenu.Item itemKey="group" startIcon={<Icon.Group />}>
        Group
      </PMenu.Item>
      {showBottomDivider && <PMenu.Divider />}
    </>
  ) : null;
