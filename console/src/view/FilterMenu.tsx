// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Dialog, Icon, Menu } from "@synnaxlabs/pluto";
import { location } from "@synnaxlabs/x";
import { type PropsWithChildren, type ReactNode } from "react";

export interface FilterMenuProps extends PropsWithChildren {}

export const FilterMenu = ({ children }: FilterMenuProps): ReactNode => (
  <Dialog.Frame>
    <Dialog.Trigger hideCaret tooltip="Filter" tooltipLocation={location.BOTTOM_RIGHT}>
      <Icon.Filter />
    </Dialog.Trigger>
    <Dialog.Dialog background={1} pack={false} bordered={false} style={style}>
      <Menu.Menu level="small" gap="small">
        {children}
      </Menu.Menu>
    </Dialog.Dialog>
  </Dialog.Frame>
);

const style = { padding: "1rem" } as const;
