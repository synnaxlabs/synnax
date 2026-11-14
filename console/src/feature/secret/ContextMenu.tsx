// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { certificate, secret } from "@synnaxlabs/client";
import { Dialog } from "@synnaxlabs/lyra/dialog";
import { Icon } from "@synnaxlabs/lyra/icon";
import { List } from "@synnaxlabs/lyra/list";
import { Menu } from "@synnaxlabs/lyra/menu";
import { Text } from "@synnaxlabs/lyra/text";
import { Access, Certificate, Secret } from "@synnaxlabs/pluto";
import { type ReactElement } from "react";

import { usedByDialogID } from "@/feature/secret/UsedBy";
import { ContextMenu } from "@/platform/context-menu";
import { Modals } from "@/platform/modals";
import { Secret as PlatformSecret } from "@/platform/secret";

const rename = (key: string): void => Text.edit(List.itemNameID(key));

/** The menu of the selected secrets. */
export const SecretsMenu = ({ keys }: Menu.ContextMenuMenuProps): ReactElement => {
  const { getItem } = List.useUtilContext<secret.Key, secret.Secret>();
  const secrets = getItem?.(keys) ?? [];
  const ids = secret.resourceID(keys);
  const canUpdate = Access.useUpdateGranted(ids);
  const canDelete = Access.useDeleteGranted(ids);
  const openReplace = PlatformSecret.useCreateModal();
  const confirm = Modals.useConfirmDelete({ type: "Secret", title: "Delete secret" });
  const { update: remove } = Secret.useDelete({
    beforeUpdate: async () => await confirm(secrets),
  });
  const single = secrets.length === 1 ? secrets[0] : undefined;
  return (
    <ContextMenu.Menu>
      {single != null && (
        <Menu.Item
          itemKey="usedBy"
          onClick={() => Dialog.open(usedByDialogID(single.key))}
        >
          <Icon.Link />
          Used by
        </Menu.Item>
      )}
      <Menu.Divider />
      {single != null && canUpdate && (
        <ContextMenu.RenameItem onClick={() => rename(single.key)} />
      )}
      <Menu.Divider />
      {single != null && canUpdate && single.source === "stored" && (
        <Menu.Item
          itemKey="replace"
          onClick={() => void openReplace({ replace: single.key })}
        >
          <Icon.Refresh />
          Replace value
        </Menu.Item>
      )}
      <Menu.Divider />
      {secrets.length > 0 && canDelete && (
        <ContextMenu.DeleteItem onClick={() => remove(secrets.map((s) => s.key))} />
      )}
      <Menu.Divider />
      <ContextMenu.ReloadConsoleItem />
    </ContextMenu.Menu>
  );
};

/** The menu of the selected certificates. */
export const CertificatesMenu = ({ keys }: Menu.ContextMenuMenuProps): ReactElement => {
  const { getItem } = List.useUtilContext<certificate.Key, certificate.Certificate>();
  const certificates = getItem?.(keys) ?? [];
  const ids = certificate.resourceID(keys);
  const canUpdate = Access.useUpdateGranted(ids);
  const canDelete = Access.useDeleteGranted(ids);
  const confirm = Modals.useConfirmDelete({
    type: "Certificate",
    title: "Delete certificate",
  });
  const { update: remove } = Certificate.useDelete({
    beforeUpdate: async () => await confirm(certificates),
  });
  const single = certificates.length === 1 ? certificates[0] : undefined;
  return (
    <ContextMenu.Menu>
      {single != null && canUpdate && (
        <ContextMenu.RenameItem onClick={() => rename(single.key)} />
      )}
      <Menu.Divider />
      {certificates.length > 0 && canDelete && (
        <ContextMenu.DeleteItem
          onClick={() => remove(certificates.map((c) => c.key))}
        />
      )}
      <Menu.Divider />
      <ContextMenu.ReloadConsoleItem />
    </ContextMenu.Menu>
  );
};
