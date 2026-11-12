// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { certificate, secret } from "@synnaxlabs/client";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Menu } from "@synnaxlabs/lyra/menu";
import { Access, Certificate, Secret } from "@synnaxlabs/pluto";
import { type ReactElement } from "react";

import { useCreateModal } from "@/feature/secret/useCreateModal";
import { ContextMenu } from "@/platform/context-menu";
import { Modals } from "@/platform/modals";

type Named = Pick<secret.Secret, "key" | "name">;

const useDeleteSecret = (): ((s: Named) => void) => {
  const confirm = Modals.useConfirmDelete({ type: "Secret", title: "Delete secret" });
  const { update } = Secret.useDelete();
  return (s) => {
    void (async () => {
      if (await confirm(s)) update(s.key);
    })();
  };
};

const useDeleteCertificate = (): ((c: Named) => void) => {
  const confirm = Modals.useConfirmDelete({
    type: "Certificate",
    title: "Delete certificate",
  });
  const { update } = Certificate.useDelete();
  return (c) => {
    void (async () => {
      if (await confirm(c)) update(c.key);
    })();
  };
};

export interface SecretMenuProps {
  secret: secret.Secret;
  onRename: (key: secret.Key) => void;
}

export const SecretMenu = ({ secret: s, onRename }: SecretMenuProps): ReactElement => {
  const id = secret.resourceID(s.key);
  const canUpdate = Access.useUpdateGranted(id);
  const canDelete = Access.useDeleteGranted(id);
  const openReplace = useCreateModal();
  const handleDelete = useDeleteSecret();
  return (
    <ContextMenu.Menu>
      {canUpdate && <ContextMenu.RenameItem onClick={() => onRename(s.key)} />}
      <Menu.Divider />
      {canUpdate && s.source === "stored" && (
        <Menu.Item itemKey="replace" onClick={() => openReplace({ replace: s.key })}>
          <Icon.Refresh />
          Replace value
        </Menu.Item>
      )}
      <Menu.Divider />
      {canDelete && <ContextMenu.DeleteItem onClick={() => handleDelete(s)} />}
      <Menu.Divider />
      <ContextMenu.ReloadConsoleItem />
    </ContextMenu.Menu>
  );
};

export interface CertificateMenuProps {
  certificate: certificate.Certificate;
  onRename: (key: certificate.Key) => void;
}

export const CertificateMenu = ({
  certificate: c,
  onRename,
}: CertificateMenuProps): ReactElement => {
  const id = certificate.resourceID(c.key);
  const canUpdate = Access.useUpdateGranted(id);
  const canDelete = Access.useDeleteGranted(id);
  const handleDelete = useDeleteCertificate();
  return (
    <ContextMenu.Menu>
      {canUpdate && <ContextMenu.RenameItem onClick={() => onRename(c.key)} />}
      <Menu.Divider />
      {canDelete && <ContextMenu.DeleteItem onClick={() => handleDelete(c)} />}
      <Menu.Divider />
      <ContextMenu.ReloadConsoleItem />
    </ContextMenu.Menu>
  );
};

export const FallbackMenu = (): ReactElement => (
  <ContextMenu.Menu>
    <ContextMenu.ReloadConsoleItem />
  </ContextMenu.Menu>
);
