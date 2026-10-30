// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/platform/search/FilterMenu.css";

import { Dialog } from "@synnaxlabs/lyra/dialog";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Menu } from "@synnaxlabs/lyra/menu";
import { type position } from "@synnaxlabs/lyra/position";
import { location } from "@synnaxlabs/x";
import { type PropsWithChildren, type ReactElement } from "react";

import { CSS } from "@/platform/css";

/** Opens a filter's select beside the menu instead of over it. */
export const FILTER_LOCATION: position.LocationPreference = {
  targetCorner: location.TOP_RIGHT,
  dialogCorner: location.TOP_LEFT,
};

export interface FilterMenuProps
  extends PropsWithChildren, Pick<Dialog.TriggerProps, "size" | "variant"> {}

/** A filter button that drops a menu of filter controls beside the search input. */
export const FilterMenu = ({
  children,
  ...triggerProps
}: FilterMenuProps): ReactElement => (
  <Dialog.Frame>
    <Dialog.Trigger
      hideCaret
      tooltip="Filter"
      tooltipLocation={location.BOTTOM_RIGHT}
      {...triggerProps}
    >
      <Icon.Filter />
    </Dialog.Trigger>
    <Dialog.Dialog
      background={1}
      pack={false}
      className={CSS.BE("search", "filter-menu")}
    >
      <Menu.Menu level="small" gap="small">
        {children}
      </Menu.Menu>
    </Dialog.Dialog>
  </Dialog.Frame>
);
