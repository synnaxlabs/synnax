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
import { Status } from "@synnaxlabs/lyra/status";
import { type ReactElement } from "react";

import { type License } from "@/server/db/schema";
import { deny } from "@/server/license/deny";
import { filename } from "@/server/license/limits";
import { post, postFile, reload, save } from "@/ui/api";
import { EditDialog } from "@/ui/licenses/EditDialog";
import { Modal } from "@/ui/modal";
import { useAction } from "@/ui/useAction";

export interface StaffActionsProps {
  license: License;
  now: Date;
}

/** StaffActions are the license actions only Synnax Labs staff can take. */
export const StaffActions = ({
  license: lic,
  now,
}: StaffActionsProps): ReactElement => {
  const floating = useAction(async () => {
    const blob = await postFile(`/api/licenses/${lic.key}/floating`);
    save(blob, filename(lic.label));
  });
  const restore = useAction(async () => {
    await post(`/api/licenses/${lic.key}/restore`);
    reload();
  });
  const enterprise = lic.edition === "enterprise";
  const error = floating.error ?? restore.error;
  return (
    <>
      {error != null && (
        <Status.Summary variant="error" level="small" message={error} />
      )}
      {lic.revokedAt != null && enterprise && (
        <Button.Button
          variant="outlined"
          onClick={restore.run}
          status={restore.loading ? "loading" : undefined}
          tooltip="Let it activate machines again while its terms allow. Its admins are emailed."
        >
          <Icon.Undo />
          Restore
        </Button.Button>
      )}
      {lic.revokedAt == null && <EditDialog license={lic} />}
      {deny(lic, now) == null && (
        <Button.Button
          variant="outlined"
          onClick={floating.run}
          status={floating.loading ? "loading" : undefined}
          tooltip="Download a license key bound to no machine, for CI runners"
        >
          <Icon.Download />
          Floating license key
        </Button.Button>
      )}
      {lic.revokedAt == null && (
        <Modal.Frame
          name={`${lic.label}.Revoke`}
          icon={<Icon.Delete />}
          trigger={
            <Dialog.Trigger variant="outlined" status="error" hideCaret>
              <Icon.Delete />
              Revoke
            </Dialog.Trigger>
          }
        >
          <Modal.Confirm
            question={`Revoke "${lic.label}"?`}
            confirm="Revoke"
            delay={1500}
            onConfirm={async () => {
              await post(`/api/licenses/${lic.key}/revoke`);
              reload();
            }}
          >
            No machine can activate it or download its license key. A Core checks its
            license offline, so a Core that already holds it keeps running as its terms
            allow. The organization's admins are emailed.
            {enterprise
              ? " Restore undoes this."
              : " A Desktop license cannot be restored."}
          </Modal.Confirm>
        </Modal.Frame>
      )}
    </>
  );
};
