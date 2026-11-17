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

import { Settings } from "@/platform/settings";
import { Session } from "@/session";

export const USAGE_DATA_URL = "https://www.synnaxlabs.com/privacy#desktop-usage-data";

/** The settings for the usage data that Synnax Desktop sends Synnax Labs. */
export const UsageDataSettings = (): ReactElement => {
  const allowed = Session.Account.useSelectAnalyticsAllowed();
  const dispatch = Session.useDispatch();
  return (
    <Settings.Card title="Usage data">
      <Settings.Entry
        label="Share usage data"
        description={
          <Flex.Box y gap="tiny" align="start">
            <Text.Text level="small" color={9}>
              Synnax Desktop sends Synnax Labs data about how you use it and the errors
              that occur.
            </Text.Text>
            <Button.Button
              variant="text"
              size="small"
              href={USAGE_DATA_URL}
              target="_blank"
            >
              <Icon.OpenExternal />
              Learn what Synnax Desktop sends
            </Button.Button>
          </Flex.Box>
        }
      >
        <Input.Switch
          aria-label="Share usage data"
          value={allowed}
          onChange={(next) => dispatch(Session.Account.setAnalyticsAllowed(next))}
        />
      </Settings.Entry>
    </Settings.Card>
  );
};
