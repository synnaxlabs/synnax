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
import { type CSSProperties, type PropsWithChildren, type ReactNode } from "react";

import { useContext } from "@/view/context";

export interface FilterMenuProps extends PropsWithChildren {}

const STYLE: CSSProperties = { padding: "1rem" };

export const FilterMenu = ({ children }: FilterMenuProps): ReactNode => {
  const { editable, isDefault } = useContext("View.FilterMenu");
  if (!editable || isDefault) return null;
  return (
    <Dialog.Frame>
      <Dialog.Trigger
        hideCaret
        tooltip="Filter"
        tooltipLocation={location.BOTTOM_RIGHT}
      >
        <Icon.Filter />
      </Dialog.Trigger>
      <Dialog.Dialog background={1} pack={false} style={STYLE}>
        <Menu.Menu level="small" gap="small">
          {children}
        </Menu.Menu>
      </Dialog.Dialog>
    </Dialog.Frame>
  );
};
