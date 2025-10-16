// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { ContextMenu, Icon } from "@synnaxlabs/pluto";
import { type ReactElement } from "react";

export interface DeleteItemProps extends Pick<ContextMenu.ItemProps, "onClick"> {}

export const DeleteItem = (props: DeleteItemProps): ReactElement => (
  <ContextMenu.Item {...props}>
    <Icon.Delete />
    Delete
  </ContextMenu.Item>
);
