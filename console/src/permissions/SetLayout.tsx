// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ReactElement } from "react";

import { type Layout } from "@/layout";
export const SET_LAYOUT_TYPE = "setPermissions";

export const setLayout = ({
  window,
  ...rest
}: Partial<Layout.State>): Layout.State => ({
  ...rest,
  key: SET_LAYOUT_TYPE,
  type: SET_LAYOUT_TYPE,
  windowKey: SET_LAYOUT_TYPE,
  icon: "Access",
  location: "modal",
  name: "Permissions.Set",
  window: {
    resizable: false,
    size: { height: 370, width: 700 },
    navTop: true,
    ...window,
  },
});

export const SetModal = (): ReactElement => {
  return <div>Set Permissions</div>;
};
