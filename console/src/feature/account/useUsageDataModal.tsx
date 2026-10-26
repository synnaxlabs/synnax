// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/feature/account/useUsageDataModal.css";

import { Button } from "@synnaxlabs/lyra/button";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Input } from "@synnaxlabs/lyra/input";
import { Modal } from "@synnaxlabs/lyra/modal";
import { Text } from "@synnaxlabs/lyra/text";
import { type ReactElement } from "react";

import { CSS } from "@/platform/css";
import { Modals } from "@/platform/modals";
import { Session } from "@/session";

const USAGE_DATA_URL = "https://docs.synnaxlabs.com/reference/desktop/usage-data";

const UsageData = (): ReactElement => {
  const allowed = Session.Account.useSelectAnalyticsAllowed();
  const dispatch = Session.useDispatch();
  return (
    <Modal.Frame className={CSS.B("account-usage-data")}>
      <Modal.Header icon={<Icon.Analyze />}>Usage data</Modal.Header>
      <Modal.Body gap="large" className={CSS.BE("account-usage-data", "body")}>
        <Input.Item label="Share usage data">
          <Input.Switch
            value={allowed}
            onChange={(next) => dispatch(Session.Account.setAnalyticsAllowed(next))}
          />
        </Input.Item>
        <Text.Text color={9}>
          Synnax Desktop sends Synnax Labs data about how you use it and the errors that
          occur.
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
      </Modal.Body>
    </Modal.Frame>
  );
};

/** Opens a dialog that turns the usage data Synnax Desktop sends on or off. */
export const useUsageDataModal = Modals.create(UsageData);
