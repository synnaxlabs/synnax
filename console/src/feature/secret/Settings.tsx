// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/feature/secret/Settings.css";

import { certificate, secret } from "@synnaxlabs/client";
import { Component } from "@synnaxlabs/lyra/component";
import { useNow } from "@synnaxlabs/lyra/hooks";
import { Input } from "@synnaxlabs/lyra/input";
import { List } from "@synnaxlabs/lyra/list";
import { Select } from "@synnaxlabs/lyra/select";
import { Text } from "@synnaxlabs/lyra/text";
import { Access, Certificate, Secret } from "@synnaxlabs/pluto";
import { TimeSpan, type TimeStamp } from "@synnaxlabs/x";
import { type ReactElement, useState } from "react";

import { CertificatesMenu, SecretsMenu } from "@/feature/secret/ContextMenu";
import { type SelectCertificate, UsedByPopover } from "@/feature/secret/UsedBy";
import { CSS } from "@/platform/css";
import { Secret as PlatformSecret } from "@/platform/secret";
import { Settings as PlatformSettings } from "@/platform/settings";

const SECRET_COLUMNS: List.Column[] = [
  { name: "Name", width: "1fr" },
  { name: "Value", width: "1.5fr" },
  { name: "Used by", width: "14rem" },
];

const CreateSecret = (): ReactElement => {
  const { created } = PlatformSettings.useCreate();
  const [name, setName] = useState("");
  const [value, setValue] = useState("");
  const { update, variant } = Secret.useCreate({
    afterSuccess: ({ data }) => created(data.key),
  });
  const save = (): void => {
    if (name.trim() === "" || variant === "loading") return;
    update({ name: name.trim(), source: "stored", value });
  };
  const onKeyDown = PlatformSettings.useCreateKeyDown(save);
  return (
    <PlatformSettings.CreateFrame>
      <List.Cell>
        <Input.Text
          value={name}
          onChange={setName}
          onKeyDown={onKeyDown}
          placeholder="Secret name"
          aria-label="Secret name"
          autoFocus
        />
      </List.Cell>
      <List.Cell>
        <Input.Text
          value={value}
          onChange={setValue}
          onKeyDown={onKeyDown}
          type="password"
          autoComplete="off"
          placeholder="Value"
          aria-label="Value"
        />
      </List.Cell>
      <List.Cell>
        <PlatformSettings.CreateActions onSave={save} status={variant} />
      </List.Cell>
    </PlatformSettings.CreateFrame>
  );
};

interface SecretItemProps extends List.ItemProps<secret.Key> {
  onSelectCertificate: SelectCertificate;
}

const SecretItem = ({ onSelectCertificate, ...rest }: SecretItemProps) => {
  const { itemKey } = rest;
  const s = List.useItem<secret.Key, secret.Secret>(itemKey);
  const canUpdate = Access.useUpdateGranted(secret.resourceID(itemKey));
  const { update: rename } = Secret.useRename();
  const [element, setElement] = useState<HTMLElement | null>(null);
  const itemClass = PlatformSettings.useItemClass(itemKey);
  if (s == null) return null;
  return (
    <Select.Item<secret.Key>
      {...rest}
      ref={setElement}
      checkable
      align="center"
      aria-label={s.name}
      className={CSS.cls(CSS.BE("secret-settings", "item"), itemClass)}
    >
      <List.Cells>
        <List.Cell>
          <Text.MaybeEditable
            id={List.itemNameID(itemKey)}
            value={s.name}
            onChange={canUpdate && ((name) => rename({ key: itemKey, name }))}
            allowDoubleClick={false}
            overflow="fade"
          />
        </List.Cell>
        <List.Cell>
          <Text.Text color={9} overflow="fade">
            {PlatformSecret.Validity.describeSecret(s)}
          </Text.Text>
        </List.Cell>
        <List.Cell>
          <UsedByPopover
            secretKey={itemKey}
            anchor={element}
            onSelectCertificate={onSelectCertificate}
          />
        </List.Cell>
      </List.Cells>
    </Select.Item>
  );
};

const secretsMenu = Component.renderProp(SecretsMenu);

export interface SecretsProps {
  /** Called when the user picks a certificate from the users of a secret. */
  onSelectCertificate: SelectCertificate;
}

/**
 * Every secret the user can read, with where its value comes from and what uses it.
 * The menu of the selected secrets renames, replaces, or deletes them.
 */
export const Secrets = ({ onSelectCertificate }: SecretsProps): ReactElement => {
  const list = Secret.useList({ pageSize: 100 });
  const canCreate = Access.useCreateGranted(secret.TYPE_RESOURCE_ID);
  return (
    <PlatformSettings.Table<secret.Key, secret.Secret>
      list={list}
      columns={SECRET_COLUMNS}
      noun="secrets"
      menu={secretsMenu}
      createLabel={canCreate ? "Create secret" : undefined}
      create={<CreateSecret />}
    >
      {({ key, ...p }) => (
        <SecretItem key={key} {...p} onSelectCertificate={onSelectCertificate} />
      )}
    </PlatformSettings.Table>
  );
};

const CERTIFICATE_COLUMNS: List.Column[] = [
  { name: "Name", width: "1fr" },
  { name: "Validity", width: "34rem" },
];

interface CertificateItemProps extends List.ItemProps<certificate.Key> {
  now: TimeStamp;
}

const CertificateItem = ({ now, ...rest }: CertificateItemProps) => {
  const { itemKey } = rest;
  const c = List.useItem<certificate.Key, certificate.Certificate>(itemKey);
  const canUpdate = Access.useUpdateGranted(certificate.resourceID(itemKey));
  const { update: rename } = Certificate.useRename();
  if (c == null) return null;
  const { text, status } = PlatformSecret.Validity.describeCertificate(c, now);
  return (
    <Select.Item<certificate.Key>
      {...rest}
      checkable
      align="center"
      aria-label={c.name}
      className={CSS.BE("secret-settings", "item")}
    >
      <List.Cells>
        <List.Cell>
          <Text.MaybeEditable
            id={List.itemNameID(itemKey)}
            value={c.name}
            onChange={canUpdate && ((name) => rename({ key: itemKey, name }))}
            allowDoubleClick={false}
            overflow="fade"
          />
        </List.Cell>
        <List.Cell>
          <Text.Text color={9} status={status} overflow="fade">
            {text}
          </Text.Text>
        </List.Cell>
      </List.Cells>
    </Select.Item>
  );
};

const nextMinute = (now: TimeStamp): TimeStamp => now.add(TimeSpan.MINUTE);

const certificatesMenu = Component.renderProp(CertificatesMenu);

export interface CertificatesProps {
  /** A certificate to select and scroll to. */
  selected?: certificate.Key;
  /** Called once the page shows the certificate in `selected`. */
  onShowSelected?: () => void;
}

/**
 * Every certificate the user can read, with when it is valid. The menu of the selected
 * certificates renames or deletes them.
 */
export const Certificates = ({
  selected,
  onShowSelected,
}: CertificatesProps): ReactElement => {
  const list = Certificate.useList({ pageSize: 100 });
  const now = useNow(nextMinute);
  return (
    <PlatformSettings.Table<certificate.Key, certificate.Certificate>
      list={list}
      columns={CERTIFICATE_COLUMNS}
      noun="certificates"
      menu={certificatesMenu}
      selected={selected}
      onShowSelected={onShowSelected}
    >
      {({ key, ...p }) => <CertificateItem key={key} {...p} now={now} />}
    </PlatformSettings.Table>
  );
};
