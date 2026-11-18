// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type access } from "@synnaxlabs/client";
import { Icon } from "@synnaxlabs/lyra/icon";

const BUILT_IN: Record<string, Icon.ReactElement> = {
  Owner: <Icon.Settings />,
  Engineer: <Icon.Safety />,
  Operator: <Icon.Control />,
  Viewer: <Icon.Visible />,
  Driver: <Icon.Rack />,
};

/** Returns the icon of a built-in role, or a generic role icon for any other role. */
export const icon = ({ name }: Pick<access.role.Role, "name">): Icon.ReactElement =>
  BUILT_IN[name] ?? <Icon.Role />;
