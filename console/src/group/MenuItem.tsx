// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Icon } from "@synnaxlabs/media";
import { Menu as PMenu } from "@synnaxlabs/pluto";

import { canGroupSelection } from "@/group/canGroupSelection";
import { type Ontology } from "@/ontology";

export interface MenuItemProps {
  selection: Ontology.TreeContextMenuProps["selection"];
}

export const MenuItem = ({ selection }: MenuItemProps) =>
  canGroupSelection(selection) ? (
    <PMenu.Item itemKey="group" startIcon={<Icon.Group />}>
      Group
    </PMenu.Item>
  ) : null;
