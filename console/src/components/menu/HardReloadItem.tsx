// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { reloadWindow } from "@synnaxlabs/drift";
import { Icon, Menu } from "@synnaxlabs/pluto";
import { type ReactElement, useCallback } from "react";
import { useDispatch } from "react-redux";

import { SHOULD_IGNORE_KEY as SHOULD_IGNORE_LINK_KEY } from "@/link/types";

export const HardReloadItem = (): ReactElement => {
  const dispatch = useDispatch();
  const handleClick = useCallback(() => {
    localStorage.setItem(SHOULD_IGNORE_LINK_KEY, "true");
    dispatch(reloadWindow({}));
  }, [dispatch]);
  return (
    <Menu.Item onClick={handleClick} size="small" itemKey="hardReload">
      <Icon.Refresh />
      Hard Reload
    </Menu.Item>
  );
};
