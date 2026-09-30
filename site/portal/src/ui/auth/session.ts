// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { navigate } from "@/ui/api";
import { type Clerk } from "@/ui/clerk";

/**
 * start makes a new sign-in's session the active one and sends the user to `target`.
 * Throws when the sign-in did not create a session.
 */
export const start = async (
  clerk: Clerk,
  sessionID: string | null,
  target: string,
): Promise<void> => {
  if (sessionID == null) throw new Error("Login did not start a session");
  await clerk.setActive({ session: sessionID });
  navigate(target);
};
