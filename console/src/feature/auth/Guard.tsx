// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import type { PropsWithChildren, ReactNode } from "react";

import { Login, useCompleteBrowserLogin } from "@/feature/auth/Login";
import { Session } from "@/session";

const CompleteBrowserLogin = (): null => {
  useCompleteBrowserLogin();
  return null;
};

export const Guard = ({ children }: PropsWithChildren): ReactNode => {
  const selected = Session.Core.useSelectIsAnySelected();
  return (
    <>
      <CompleteBrowserLogin />
      {selected ? children : <Login />}
    </>
  );
};
