// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Button } from "@synnaxlabs/lyra/button";
import { Dialog } from "@synnaxlabs/lyra/dialog";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Text } from "@synnaxlabs/lyra/text";
import { type ReactElement, useCallback } from "react";

import { type Activation } from "@/server/db/schema";
import { post, reload } from "@/ui/api";
import { machineName } from "@/ui/format";
import * as Modal from "@/ui/Modal";
import { useAction } from "@/ui/useAction";

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
    <LogOutContent activation={activation} />
  </Modal.Frame>
);

const LogOutContent = ({ activation }: { activation: Activation }): ReactElement => {
  const { close } = Dialog.useContext();
  const action = useAction(
    useCallback(async () => {
      await post(`/api/activations/${activation.key}/unlink`);
      close();
      await reload();
    }, [activation.key, close]),
  );
  return (
    <>
      <Modal.Body gap="small">
        <Text.Text level="h4" weight={450}>
          Log out {machineName(activation)}?
        </Text.Text>
        <Text.Text level="p" color={9}>
          Synnax Desktop on that machine stops renewing its license and asks you to log
          in again.
        </Text.Text>
      </Modal.Body>
      <Modal.Footer error={action.error} hint="Press and hold Log out to confirm">
        <Modal.Cancel />
        <Button.Button
          variant="filled"
          status={action.loading ? "loading" : "error"}
          onClick={action.run}
          onClickDelay={1000}
        >
          Log out
        </Button.Button>
      </Modal.Footer>
    </>
  );
};
