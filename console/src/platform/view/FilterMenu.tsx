// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Dialog } from "@synnaxlabs/lyra/dialog";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Menu } from "@synnaxlabs/lyra/menu";
import { location } from "@synnaxlabs/x";
import { type PropsWithChildren, type ReactNode } from "react";

import { useContext } from "@/platform/view/context";

export interface FilterMenuProps extends PropsWithChildren {}

export const FilterMenu = ({ children }: FilterMenuProps): ReactNode => {
  const { editable } = useContext("View.FilterMenu");
  if (!editable) return null;
  return (
    <Dialog.Frame>
      <Dialog.Trigger
        hideCaret
        variant="text"
        size="medium"
        textColor={9}
        gap="small"
        tooltip="Filter"
        tooltipLocation={location.BOTTOM_RIGHT}
      >
        <Icon.Filter />
        Filter
      </Dialog.Trigger>
      <Menu.Dialog>
        <Menu.Menu gap="small">{children}</Menu.Menu>
      </Menu.Dialog>
    </Dialog.Frame>
  );
};
