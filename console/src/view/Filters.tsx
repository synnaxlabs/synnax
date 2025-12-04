// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Dialog, Icon } from "@synnaxlabs/pluto";
import { type PropsWithChildren, type ReactNode } from "react";

export interface FiltersProps extends PropsWithChildren {
  dialog?: boolean;
}

export const Filters = ({ children, dialog = false }: FiltersProps): ReactNode => {
  if (dialog)
    return (
      <Dialog.Frame>
        <Dialog.Trigger hideCaret tooltip="Filter">
          <Icon.Filter />
        </Dialog.Trigger>
        <Dialog.Dialog
          background={1}
          bordered={false}
          pack={false}
          style={{ padding: "1rem" }}
        >
          {children}
        </Dialog.Dialog>
      </Dialog.Frame>
    );
  return children;
};
