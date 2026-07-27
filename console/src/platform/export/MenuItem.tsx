// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ontology } from "@synnaxlabs/client";
import { type ReactElement } from "react";

import {
  ContextMenuItem,
  type ContextMenuItemProps,
} from "@/platform/export/ContextMenuItem";
import { use } from "@/platform/export/use";

export interface MenuItemProps extends Omit<ContextMenuItemProps, "onClick" | "id"> {
  /** Resolves the ontology ID to export, evaluated when the item is clicked. */
  getID: () => ontology.ID;
}

/**
 * A context-menu item wired to export the resource identified by getID. Prefer this over
 * ContextMenuItem, the presentational primitive that custom flows (e.g. multi-resource
 * or directory exports) use with their own onClick.
 */
export const MenuItem = ({ getID, ...rest }: MenuItemProps): ReactElement => {
  const handleExport = use();
  return <ContextMenuItem onClick={() => handleExport(getID())} {...rest} />;
};
