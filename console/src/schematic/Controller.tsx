// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Control, User } from "@synnaxlabs/pluto";
import { type ReactElement, useCallback } from "react";
import { useDispatch } from "react-redux";

import { Layout } from "@/layout";
import { setControlStatus } from "@/schematic/slice";

export interface ControllerProps extends Omit<
  Control.ControllerProps,
  "name" | "onStatusChange"
> {
  resourceKey: string;
}

export const Controller = ({ resourceKey, ...rest }: ControllerProps): ReactElement => {
  const name = Layout.useSelectRequiredName(resourceKey);
  const dispatch = useDispatch();
  const { data: user } = User.useRetrieve({}, { addStatusOnFailure: false });
  const username = user?.username ?? "";
  const controlName = username.length > 0 ? `${name} (${username})` : name;
  const handleStatusChange = useCallback(
    (next: Control.Status) =>
      dispatch(setControlStatus({ key: resourceKey, control: next })),
    [dispatch, resourceKey],
  );
  return (
    <Control.Controller
      onStatusChange={handleStatusChange}
      name={controlName}
      {...rest}
    />
  );
};
