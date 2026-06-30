// Copyright 2026 Synnax Labs, Inc.
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

import { Link } from "@/service/link";

export const ReloadConsoleItem = (): ReactElement => {
  const dispatch = useDispatch();
  const handleClick = useCallback(() => {
    Link.markNextIgnored();
    dispatch(reloadWindow({}));
  }, [dispatch]);
  return (
    <Menu.Item onClick={handleClick} itemKey="hardReload">
      <Icon.Refresh />
      Reload Console
    </Menu.Item>
  );
};
