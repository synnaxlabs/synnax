// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Drift } from "@synnaxlabs/drift";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Menu } from "@synnaxlabs/lyra/menu";
import type { ReactElement } from "react";

import { Session } from "@/session";

export const ReloadConsoleItem = (): ReactElement => {
  const dispatch = Session.useDispatch();
  const reload = () => dispatch(Drift.reloadWindow({}));
  // A change saves after a debounce, so the reload saves first. A failed save still
  // reloads, as the Session reports the failure itself.
  const handleClick = () => void Session.Persist.flush(dispatch).then(reload, reload);
  return (
    <Menu.Item onClick={handleClick} itemKey="hardReload">
      <Icon.Refresh />
      Reload Console
    </Menu.Item>
  );
};
