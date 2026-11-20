// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Control, Schematic, User } from "@synnaxlabs/pluto";
import { type PropsWithChildren, type ReactElement } from "react";

import { Framer } from "@/platform/framer";
import { Session } from "@/session";

export interface ControllerProps extends PropsWithChildren {}

export const Controller = ({ children }: ControllerProps): ReactElement => {
  const key = Schematic.useKey();
  const authority = Session.Schematic.useSelectAuthority();
  const { isCurrentlyEditable } = Session.Schematic.useSelectEditable();
  const canCommand = Framer.useCanCommand();
  const name = Schematic.useName();
  const dispatch = Session.useDispatch();
  const { data: userName } = User.useResultName({});
  const controlName = userName != null ? `${name} (${userName})` : name;
  const handleStatusChange = (status: Control.Status) =>
    dispatch(Session.Schematic.setControlStatus({ key, status }));
  return (
    <Control.Controller
      onStatusChange={handleStatusChange}
      name={controlName}
      authority={authority}
      disabled={isCurrentlyEditable || !canCommand}
    >
      {children}
    </Control.Controller>
  );
};
Controller.displayName = "Schematic.Controller";
