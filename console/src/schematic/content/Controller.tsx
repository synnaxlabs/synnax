// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Control, Schematic, User } from "@synnaxlabs/pluto";
import { memo, type ReactElement } from "react";

import { Session } from "@/schematic/session";

export interface ControllerProps extends Omit<
  Control.ControllerProps,
  "name" | "onStatusChange"
> {}

export const Controller = memo((props: ControllerProps): ReactElement => {
  const name = Schematic.useSelectName({});
  const username = User.useRetrieve({}, { addStatusOnFailure: false }).data?.username;
  const controlName = username != null ? `${name} (${username})` : name;
  const handleStatusChange = Session.useSetControlStatus();
  return (
    <Control.Controller
      onStatusChange={handleStatusChange}
      name={controlName}
      {...props}
    />
  );
});
Controller.displayName = "Controller";
