// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Control, Schematic, User } from "@synnaxlabs/pluto";
import { type ReactElement, useCallback } from "react";
import { useDispatch } from "react-redux";

import { Layout } from "@/layout";
import { useSelectAuthority } from "@/schematic/selectors";
import { setControlStatus } from "@/schematic/slice";

export interface ControllerProps extends Omit<
  Control.ControllerProps,
  "name" | "onStatusChange"
> {}

export const Controller = (props: ControllerProps): ReactElement => {
  const key = Schematic.useKey();
  const authority = useSelectAuthority(key);
  const name = Layout.useSelectRequiredName(key);
  const dispatch = useDispatch();
  const { data: user } = User.useRetrieve({}, { addStatusOnFailure: false });
  const username = user?.username ?? "";
  const controlName = username.length > 0 ? `${name} (${username})` : name;
  const handleStatusChange = useCallback(
    (next: Control.Status) => dispatch(setControlStatus({ key, control: next })),
    [dispatch, key],
  );
  return (
    <Control.Controller
      onStatusChange={handleStatusChange}
      name={controlName}
      authority={authority}
      {...props}
    />
  );
};
