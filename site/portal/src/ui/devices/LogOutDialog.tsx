// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Dialog } from "@synnaxlabs/lyra/dialog";
import { Icon } from "@synnaxlabs/lyra/icon";
import { type ReactElement } from "react";

import { type Activation } from "@/server/db/schema";
import { post, reload } from "@/ui/api";
import { machineName } from "@/ui/format";
import { Modal } from "@/ui/modal";

/** LogOutDialog logs a Desktop machine out of the account, after a confirmation. */
export const LogOutDialog = ({
  activation,
}: {
  activation: Activation;
}): ReactElement => (
  <Modal.Frame
    name="Log out this machine"
    icon={<Icon.Logout />}
    trigger={
      <Dialog.Trigger variant="text" size="small" hideCaret status="error">
        Log out
      </Dialog.Trigger>
    }
  >
    <Modal.Confirm
      question={`Log out ${machineName(activation)}?`}
      confirm="Log out"
      onConfirm={async () => {
        await post(`/api/activations/${activation.key}/unlink`);
        reload();
      }}
    >
      Synnax Desktop on that machine stops renewing its license and asks you to log in
      again.
    </Modal.Confirm>
  </Modal.Frame>
);
