// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Icon } from "@synnaxlabs/lyra/icon";
import { Menu } from "@synnaxlabs/lyra/menu";

export interface CreateMenuItemProps extends Omit<Menu.ItemProps, "startIcon"> {}

const CreateTaskIcon = Icon.createFromPath(
  "task-create",
  "M11.27,11.48c0.05,0.32 0.2,0.61 0.42,0.83c0,0 0,0 0,0c0.16,0.16 0.36,0.29 0.59,0.37c0.15,0.05 0.3,0.07 0.47,0.07h6.75c0.13,0 0.26,0.05 0.36,0.15c0.08,0.09 0.13,0.2 0.14,0.31c-0.11,4.87 -4.1,8.79 -9,8.79c-0.5,0 -0.9,-0.4 -0.9,-0.9v-6.54l-0.08,-0.05c-0.47,-0.31 -0.77,-0.82 -0.82,-1.38l0,-0.13c0,-0.73 0.43,-1.36 1.06,-1.64c0,0 0,0 0,0c0.02,-0.01 0.05,-0.02 0.07,-0.03c0.17,-0.07 0.35,-0.11 0.54,-0.12c0.17,0.02 0.32,0.12 0.4,0.28zM10.79,5.81c-0.03,0 -0.05,0 -0.08,0c-0.19,0 -0.23,0.01 -0.47,0.03v0c-0.02,0 -0.04,0 -0.07,0.01c-3.33,0.38 -5.96,3.01 -6.33,6.34c-0.38,3.33 1.59,6.48 4.75,7.6c0.47,0.17 0.71,0.68 0.55,1.15c-0.17,0.47 -0.68,0.71 -1.15,0.55c-3.95,-1.4 -6.42,-5.34 -5.94,-9.5c0.47,-4.16 3.76,-7.45 7.92,-7.93c0.26,-0.03 0.52,-0.05 0.78,-0.06c0.13,0 0.25,0.05 0.34,0.14c0.1,0.09 0.15,0.22 0.15,0.36v0.8c0,0.26 -0.2,0.48 -0.46,0.5zM10.83,9.4c-0.04,0 -0.07,0 -0.11,0.01c-0.24,0 -0.25,0.02 -0.47,0.07v0c-0.01,0 -0.01,0 -0.02,0c-1.38,0.3 -2.45,1.37 -2.75,2.75c-0.3,1.38 0.23,2.8 1.36,3.65c0.4,0.3 0.48,0.86 0.18,1.26c-0.3,0.4 -0.86,0.48 -1.26,0.18c-1.69,-1.27 -2.49,-3.4 -2.04,-5.47c0.45,-2.06 2.06,-3.68 4.13,-4.13c0.31,-0.07 0.61,-0.11 0.92,-0.12c0.12,0 0.24,0.05 0.33,0.14c0.1,0.09 0.16,0.23 0.16,0.36v0.81c0,0.25 -0.18,0.45 -0.42,0.49zM13,7.75v-2.5h3.25v-3.25h2.5v3.25h3.25v2.5h-3.25v3.25h-2.5v-3.25z",
);

export const CreateMenuItem = ({ children, ...rest }: CreateMenuItemProps) => (
  <Menu.Item {...rest}>
    <CreateTaskIcon />
    {children}
  </Menu.Item>
);
