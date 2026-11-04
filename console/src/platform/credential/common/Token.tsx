// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Button } from "@synnaxlabs/lyra/button";
import { Flex } from "@synnaxlabs/lyra/flex";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Input } from "@synnaxlabs/lyra/input";
import { Text } from "@synnaxlabs/lyra/text";
import { type ReactElement } from "react";

import { Clipboard } from "@/platform/clipboard";

export interface TokenProps {
  /** A new secret, shown once. */
  token: string;
  /** What the secret is, such as "API key". */
  label: string;
}

/** Shows a new secret with a copy button and a warning that it is not shown again. */
export const Token = ({ token, label }: TokenProps): ReactElement => {
  const copy = Clipboard.useCopy();
  return (
    <Flex.Box y gap="tiny">
      <Text.Text level="small" status="warning">
        Copy this {label} now. You won't see it again.
      </Text.Text>
      <Flex.Box x align="center">
        <Input.Text value={token} onChange={() => {}} full="x" disabled />
        <Button.Button
          variant="text"
          tooltip={`Copy ${label}`}
          onClick={() => copy(token, label)}
        >
          <Icon.Copy />
        </Button.Button>
      </Flex.Box>
    </Flex.Box>
  );
};
