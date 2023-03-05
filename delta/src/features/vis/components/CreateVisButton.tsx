// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Icon } from "@synnaxlabs/media";
import { Button, ButtonProps } from "@synnaxlabs/pluto";
import { useDispatch } from "react-redux";

import { createLineVis } from "./line/types";
import { VisToolbar } from "./VisToolbar";

import { useLayoutPlacer, setNavdrawerVisible } from "@/features/layout";

export interface CreateVisButtonProps
  extends Omit<ButtonProps, "startIcon" | "endIcon" | "children"> {}

export const CreateVisButton = (props: CreateVisButtonProps): JSX.Element => {
  const newLayout = useLayoutPlacer();
  const dispatch = useDispatch();
  const handleClick = (): void => {
    newLayout(createLineVis({}));
    dispatch(setNavdrawerVisible({ key: VisToolbar.key, value: true }));
  };
  return (
    <Button
      {...props}
      startIcon={<Icon.Visualize />}
      onClick={handleClick}
      size="large"
    >
      Create a Visualization
    </Button>
  );
};
