// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Button } from "@synnaxlabs/pluto";
import { HiLightningBolt } from "react-icons/hi";
import { useDispatch } from "react-redux";

import { setWarpMode, useSelectWarpMode } from "../store";

import { setNavdrawerEntryState } from "@/features/layout";

export const WarpModeToggle = (): JSX.Element => {
  const checked = useSelectWarpMode();
  const dispatch = useDispatch();

  const handleClick = (): void => {
    dispatch(
      setNavdrawerEntryState({
        location: "bottom",
        state: {
          menuItems: checked ? ["visualization"] : [],
          activeItem: checked ? "visualization" : null,
        },
      })
    );
    dispatch(setWarpMode());
  };

  return (
    <Button.IconOnlyToggle checked={checked} onClick={handleClick}>
      <HiLightningBolt />
    </Button.IconOnlyToggle>
  );
};
