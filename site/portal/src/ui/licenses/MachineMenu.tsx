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
import { Menu } from "@synnaxlabs/lyra/menu";
import { Status } from "@synnaxlabs/lyra/status";
import { type ReactElement, useState } from "react";

import { type Activation } from "@/server/db/schema";
import { filename } from "@/server/license/limits";
import { post, postFile, reload, save } from "@/ui/api";
import { machineName } from "@/ui/format";
import { RenameDialog } from "@/ui/licenses/RenameDialog";
import * as Modal from "@/ui/Modal";
import { useAction } from "@/ui/useAction";

export interface MachineMenuProps {
  activation: Activation;
  label: string;
}

/** MachineMenu holds the actions on a machine that holds a seat. */
export const MachineMenu = ({ activation, label }: MachineMenuProps): ReactElement => {
  const [releasing, setReleasing] = useState(false);
  const [renaming, setRenaming] = useState(false);
  const download = useAction(async () => {
    const blob = await postFile(`/api/activations/${activation.key}/download`);
    save(blob, filename(label));
  });
  const release = (): void => setReleasing(true);
  const rename = (): void => setRenaming(true);
  return (
    <>
      {download.error != null && (
        <Status.Summary variant="error" level="small" message={download.error} />
      )}
      <Dialog.Frame variant="floating" location={{ x: "right", y: "bottom" }}>
        <Dialog.Trigger
          variant="text"
          size="small"
          hideCaret
          aria-label="Machine actions"
        >
          <Icon.KebabMenu />
        </Dialog.Trigger>
        <Dialog.Dialog bordered rounded background={1} className="portal-machine-menu">
          <Menu.Menu
            level="small"
            onChange={{ download: download.run, rename, release }}
          >
            <Menu.Item itemKey="download">
              <Icon.Download />
              Download license key
            </Menu.Item>
            <Menu.Item itemKey="rename">
              <Icon.Rename />
              Rename
            </Menu.Item>
            <Menu.Item itemKey="release" status="error">
              <Icon.Release />
              Release seat
            </Menu.Item>
          </Menu.Menu>
        </Dialog.Dialog>
      </Dialog.Frame>
      <Modal.Frame
        name="Release this seat"
        icon={<Icon.Release />}
        visible={releasing}
        onVisibleChange={setReleasing}
      >
        <Modal.Confirm
          question={`Release the seat held by ${machineName(activation)}?`}
          confirm="Release"
          onConfirm={async () => {
            await post(`/api/activations/${activation.key}/release`);
            reload();
          }}
        >
          The Core on that machine loses its license at its next check. Activate it
          again to give it a new license key.
        </Modal.Confirm>
      </Modal.Frame>
      <RenameDialog
        activation={activation}
        visible={renaming}
        onVisibleChange={setRenaming}
      />
    </>
  );
};
