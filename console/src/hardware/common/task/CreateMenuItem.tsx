// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Icon } from "@synnaxlabs/media";
import { Icon as PIcon, Menu } from "@synnaxlabs/pluto";

const CreateTaskIcon = () => (
  <PIcon.Create>
    <Icon.Task />
  </PIcon.Create>
);

export interface CreateMenuItemProps extends Omit<Menu.ItemProps, "startIcon"> {}

export const CreateMenuItem = (props: CreateMenuItemProps) => (
  <Menu.Item {...props} startIcon={<CreateTaskIcon />} />
);
