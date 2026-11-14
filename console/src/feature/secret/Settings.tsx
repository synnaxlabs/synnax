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
import { Button } from "@synnaxlabs/lyra/button";
import { Component } from "@synnaxlabs/lyra/component";
import { Flex } from "@synnaxlabs/lyra/flex";
import { useNow } from "@synnaxlabs/lyra/hooks";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Input } from "@synnaxlabs/lyra/input";
import { List } from "@synnaxlabs/lyra/list";
import { Menu } from "@synnaxlabs/lyra/menu";
import { Select } from "@synnaxlabs/lyra/select";
import { Text } from "@synnaxlabs/lyra/text";
import { Access, Certificate, type Flux, Secret } from "@synnaxlabs/pluto";
import { TimeSpan, type TimeStamp } from "@synnaxlabs/x";
import { type ReactElement, type ReactNode, useState } from "react";

import { CertificatesMenu, SecretsMenu } from "@/feature/secret/ContextMenu";
import { type SelectCertificate, UsedByPopover } from "@/feature/secret/UsedBy";
import { CSS } from "@/platform/css";
import { Secret as PlatformSecret } from "@/platform/secret";
import { Settings as PlatformSettings } from "@/platform/settings";

interface TableProps<K extends string, E extends { key: K }> {
  list: Flux.UseListReturn<K, E>;
  columns: List.Column[];
  noun: string;
  action?: ReactNode;
  menu: Menu.ContextMenuProps["menu"];
  children: List.ItemsProps<K>["children"];
}

const Table = <K extends string, E extends { key: K }>({
  list,
  columns,
  noun,
  action,
  menu,
  children,
}: TableProps<K, E>): ReactElement => {
  const { data, getItem, subscribe, more, search, answered } = list;
  const [term, setTerm] = useState("");
  const [selected, setSelected] = useState<K[]>([]);
  const menuProps = Menu.useContextMenu();
  const handleSearch = (next: string): void => {
    setTerm(next);
    search(next);
  };
  return (
    <Select.Frame<K, E>
      multiple
      data={data}
      getItem={getItem}
      subscribe={subscribe}
      onFetchMore={more}
      value={selected}
      onChange={setSelected}
    >
      <Menu.ContextMenu menu={menu} {...menuProps} />
      <Flex.Box y gap="medium">
        <Flex.Box x align="center" justify="between" gap="medium">
          <Input.Text
            value={term}
            onChange={handleSearch}
            placeholder={`Search ${noun}`}
            startContent={<Icon.Search />}
            className={CSS.BE("secret-settings", "search")}
          />
          {action}
        </Flex.Box>
        <PlatformSettings.Card flush>
          <List.Columns columns={columns}>
            <List.Headings checkable />
            <List.Scroll onContextMenu={menuProps.open}>
              <List.Items<K>
                emptyContent={
                  answered && (
                    <Flex.Box center className={CSS.BE("secret-settings", "empty")}>
                      <Text.Text color={9}>No {noun}</Text.Text>
                    </Flex.Box>
                  )
                }
              >
                {children}
              </List.Items>
            </List.Scroll>
          </List.Columns>
        </PlatformSettings.Card>
      </Flex.Box>
    </Select.Frame>
  );
};

const SECRET_COLUMNS: List.Column[] = [
  { name: "Name", width: "1fr" },
  { name: "Value", width: "34rem" },
  { name: "Used by", width: "14rem" },
];

interface SecretItemProps extends List.ItemProps<secret.Key> {
  onSelectCertificate: SelectCertificate;
}

const SecretItem = ({ onSelectCertificate, ...rest }: SecretItemProps) => {
  const { itemKey } = rest;
  const s = List.useItem<secret.Key, secret.Secret>(itemKey);
  const canUpdate = Access.useUpdateGranted(secret.resourceID(itemKey));
  const { update: rename } = Secret.useRename();
  const [element, setElement] = useState<HTMLElement | null>(null);
  if (s == null) return null;
  return (
    <Select.Item<secret.Key>
      {...rest}
      ref={setElement}
      checkable
      align="center"
      aria-label={s.name}
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
  const openCreate = PlatformSecret.useCreateModal();
  const canCreate = Access.useCreateGranted(secret.TYPE_RESOURCE_ID);
  return (
    <Table<secret.Key, secret.Secret>
      list={list}
      columns={SECRET_COLUMNS}
      noun="secrets"
      menu={secretsMenu}
      action={
        canCreate && (
          <Button.Button variant="outlined" onClick={() => void openCreate()}>
            <Icon.Add />
            Create secret
          </Button.Button>
        )
      }
    >
      {({ key, ...p }) => (
        <SecretItem key={key} {...p} onSelectCertificate={onSelectCertificate} />
      )}
    </Table>
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

/**
 * Every certificate the user can read, with when it is valid. The menu of the selected
 * certificates renames or deletes them.
 */
export const Certificates = (): ReactElement => {
  const list = Certificate.useList({ pageSize: 100 });
  const now = useNow(nextMinute);
  return (
    <Table<certificate.Key, certificate.Certificate>
      list={list}
      columns={CERTIFICATE_COLUMNS}
      noun="certificates"
      menu={certificatesMenu}
    >
      {({ key, ...p }) => <CertificateItem key={key} {...p} now={now} />}
    </Table>
  );
};
