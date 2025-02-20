// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { reloadWindow } from "@synnaxlabs/drift";
import { Icon } from "@synnaxlabs/media";
import { Menu } from "@synnaxlabs/pluto";
import { useCallback } from "react";
import { useDispatch } from "react-redux";

export const HardReloadItem = () => {
  const dispatch = useDispatch();
  const handleClick = useCallback(() => dispatch(reloadWindow({})), [dispatch]);
  return (
    <Menu.Item
      onClick={handleClick}
      startIcon={<Icon.Refresh />}
      size="small"
      itemKey="hardReload"
    >
      Hard Reload
    </Menu.Item>
  );
};
