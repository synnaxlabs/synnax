// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Control, Schematic, User } from "@synnaxlabs/pluto";
import { memo, type PropsWithChildren, type ReactElement, useCallback } from "react";
import { useDispatch } from "react-redux";

import { Session } from "@/schematic/session";

export interface ControllerProps extends PropsWithChildren {}

export const Controller = memo(({ children }: ControllerProps): ReactElement => {
  const key = Schematic.useKey();
  const authority = Session.useSelectAuthority();
  const name = Schematic.useSelectName();
  const dispatch = useDispatch();
  const { data: user } = User.useRetrieve({}, { addStatusOnFailure: false });
  const username = user?.username ?? "";
  const controlName = username.length > 0 ? `${name} (${username})` : name;
  const handleStatusChange = useCallback(
    (status: Control.Status) => dispatch(Session.setControlStatus({ key, status })),
    [dispatch, key],
  );
  return (
    <Control.Controller
      onStatusChange={handleStatusChange}
      name={controlName}
      authority={authority}
    >
      {children}
    </Control.Controller>
  );
});
Controller.displayName = "Schematic.Controller";
