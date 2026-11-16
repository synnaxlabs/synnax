// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Text } from "@synnaxlabs/lyra/text";
import { type ReactElement } from "react";

/**
 * ReleaseNotes tells the visitor that every account gets release notes by email. It
 * sits on both the sign-up and the login card, since a Google or Microsoft login can
 * make an account.
 */
export const ReleaseNotes = (): ReactElement => (
  <Text.Text level="small" color={9}>
    We email you our release notes. You can unsubscribe at any time.
  </Text.Text>
);
