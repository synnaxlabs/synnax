// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Icon } from "@synnaxlabs/lyra/icon";
import { Note } from "@synnaxlabs/lyra/note";
import { Text } from "@synnaxlabs/lyra/text";
import type { ReactElement } from "react";

export interface WarningProps {
  /** cause tells why the symbol cannot be drawn, and sets the text of the warning. */
  cause: "missing" | "failed" | "invalid";
  className?: string;
}

const TEXTS: Record<WarningProps["cause"], string> = {
  missing: "Missing symbol",
  failed: "Failed to load symbol",
  invalid: "Invalid symbol",
};

/** Warning takes the place of a symbol that a node cannot draw. */
export const Warning = ({ cause, className }: WarningProps): ReactElement => (
  <Note.Note variant="warning" className={className}>
    <Text.Text level="p" status="warning">
      <Icon.Warning />
      {TEXTS[cause]}
    </Text.Text>
  </Note.Note>
);
