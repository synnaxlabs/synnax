// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { certificate, secret } from "@synnaxlabs/client";
import { Flex } from "@synnaxlabs/lyra/flex";
import { useNow } from "@synnaxlabs/lyra/hooks";
import { Icon } from "@synnaxlabs/lyra/icon";
import { List } from "@synnaxlabs/lyra/list";
import { Menu } from "@synnaxlabs/lyra/menu";
import { Select } from "@synnaxlabs/lyra/select";
import { Status } from "@synnaxlabs/lyra/status";
import { Text } from "@synnaxlabs/lyra/text";
import { Access, Certificate, Secret } from "@synnaxlabs/pluto";
import { TimeSpan, type TimeStamp } from "@synnaxlabs/x";
import { type ReactElement, type ReactNode, useState } from "react";

import {
  CertificateMenu,
  FallbackMenu,
  SecretMenu,
} from "@/feature/secret/ContextMenu";
import { type Lists as ListsData, useLists } from "@/feature/secret/useLists";
import { Validity } from "@/feature/secret/validity";
import { CSS } from "@/platform/css";

const SecretItem = (props: List.ItemProps<secret.Key>) => {
  const { itemKey } = props;
  const s = List.useItem<secret.Key, secret.Secret>(itemKey);
  const canUpdate = Access.useUpdateGranted(secret.resourceID(itemKey));
  const { update: rename } = Secret.useRename();
  if (s == null) return null;
  return (
    <Select.Item
      {...props}
      className={CSS.BE("credentials", "item")}
      align="center"
      gap="medium"
    >
      <Icon.Secret className={CSS.BE("credentials", "icon")} />
      <Flex.Box y empty grow className={CSS.BE("credentials", "text")}>
        <Text.MaybeEditable
          id={List.itemNameID(itemKey)}
          value={s.name}
          onChange={canUpdate ? (name) => rename({ key: itemKey, name }) : undefined}
          allowDoubleClick={false}
          overflow="fade"
        />
        <Text.Text level="tiny" color={9} overflow="fade">
          {Validity.describeSecret(s)}
        </Text.Text>
      </Flex.Box>
    </Select.Item>
  );
};

interface CertificateItemProps extends List.ItemProps<certificate.Key> {
  now: TimeStamp;
}

const CertificateItem = ({ now, ...rest }: CertificateItemProps) => {
  const { itemKey } = rest;
  const c = List.useItem<certificate.Key, certificate.Certificate>(itemKey);
  const canUpdate = Access.useUpdateGranted(certificate.resourceID(itemKey));
  const { update: rename } = Certificate.useRename();
  if (c == null) return null;
  const { text, status } = Validity.describeCertificate(c, now);
  return (
    <Select.Item
      {...rest}
      className={CSS.BE("credentials", "item")}
      align="center"
      gap="medium"
    >
      <Icon.Certificate className={CSS.BE("credentials", "icon")} />
      <Flex.Box y empty grow className={CSS.BE("credentials", "text")}>
        <Text.MaybeEditable
          id={List.itemNameID(itemKey)}
          value={c.name}
          onChange={canUpdate ? (name) => rename({ key: itemKey, name }) : undefined}
          allowDoubleClick={false}
          overflow="fade"
        />
        <Text.Text level="tiny" color={9} status={status} overflow="fade">
          {text}
        </Text.Text>
      </Flex.Box>
    </Select.Item>
  );
};

interface SectionProps {
  title: string;
  children: ReactNode;
}

const Section = ({ title, children }: SectionProps): ReactElement => (
  <Flex.Box y empty className={CSS.BE("credentials", "section")}>
    <Text.Text level="small" color={9} className={CSS.BE("credentials", "heading")}>
      {title}
    </Text.Text>
    {children}
  </Flex.Box>
);

const renameByKey = (key: string): void => Text.edit(List.itemNameID(key));

const EMPTY_SECRETS = (
  <Text.Text level="small" color={8} className={CSS.BE("credentials", "empty")}>
    No secrets
  </Text.Text>
);

const EMPTY_CERTIFICATES = (
  <Text.Text level="small" color={8} className={CSS.BE("credentials", "empty")}>
    No certificates
  </Text.Text>
);

const nextMinute = (now: TimeStamp): TimeStamp => now.add(TimeSpan.MINUTE);

type ListState = Pick<NonNullable<ListsData["secrets"]>, "answered" | "status">;

const emptyContent = (list: ListState, empty: ReactElement): ReactNode => {
  if (list.status.variant === "error")
    return (
      <Status.Summary
        status={list.status}
        level="small"
        className={CSS.BE("credentials", "empty")}
      />
    );
  return list.answered && empty;
};

interface ListSectionProps<L> {
  list: NonNullable<L>;
  selected: string | undefined;
  onSelect: (key: string | null) => void;
}

const SecretSection = ({
  list,
  selected,
  onSelect,
}: ListSectionProps<ListsData["secrets"]>): ReactElement => {
  const menu = Menu.useContextMenu();
  const renderMenu = ({ keys }: Menu.ContextMenuMenuProps) => {
    const s = keys.length === 1 ? list.getItem(keys[0]) : undefined;
    if (s == null) return <FallbackMenu />;
    return <SecretMenu secret={s} onRename={renameByKey} />;
  };
  return (
    <Section title="Secrets">
      <Menu.ContextMenu menu={renderMenu} {...menu} />
      <Select.Frame<secret.Key, secret.Secret>
        data={list.data}
        getItem={list.getItem}
        subscribe={list.subscribe}
        onFetchMore={list.more}
        value={selected}
        onChange={onSelect}
        allowNone
      >
        <List.Scroll
          onContextMenu={menu.open}
          className={CSS.BE("credentials", "scroll")}
        >
          <List.Items<secret.Key, secret.Secret>
            emptyContent={emptyContent(list, EMPTY_SECRETS)}
          >
            {({ key, ...p }) => <SecretItem key={key} {...p} />}
          </List.Items>
        </List.Scroll>
      </Select.Frame>
    </Section>
  );
};

const CertificateSection = ({
  list,
  selected,
  onSelect,
}: ListSectionProps<ListsData["certificates"]>): ReactElement => {
  const now = useNow(nextMinute);
  const menu = Menu.useContextMenu();
  const renderMenu = ({ keys }: Menu.ContextMenuMenuProps) => {
    const c = keys.length === 1 ? list.getItem(keys[0]) : undefined;
    if (c == null) return <FallbackMenu />;
    return <CertificateMenu certificate={c} onRename={renameByKey} />;
  };
  return (
    <Section title="Certificates">
      <Menu.ContextMenu menu={renderMenu} {...menu} />
      <Select.Frame<certificate.Key, certificate.Certificate>
        data={list.data}
        getItem={list.getItem}
        subscribe={list.subscribe}
        onFetchMore={list.more}
        value={selected}
        onChange={onSelect}
        allowNone
      >
        <List.Scroll
          onContextMenu={menu.open}
          className={CSS.BE("credentials", "scroll")}
        >
          <List.Items<certificate.Key, certificate.Certificate>
            emptyContent={emptyContent(list, EMPTY_CERTIFICATES)}
          >
            {({ key, ...p }) => <CertificateItem key={key} {...p} now={now} />}
          </List.Items>
        </List.Scroll>
      </Select.Frame>
    </Section>
  );
};

const isEmpty = (list: { answered: boolean; data: unknown[] } | null): boolean =>
  list == null || (list.answered && list.data.length === 0);

export interface ListsProps {
  emptyContent: ReactNode;
}

interface BodyProps extends ListsProps, ListsData {}

const Body = ({ secrets, certificates, emptyContent }: BodyProps): ReactElement => {
  const [selected, setSelected] = useState<string | undefined>(undefined);
  const handleSelect = (key: string | null) => setSelected(key ?? undefined);
  if (isEmpty(secrets) && isEmpty(certificates)) return <>{emptyContent}</>;
  return (
    <Flex.Box y empty full className={CSS.BE("credentials", "lists")}>
      {secrets != null && (
        <SecretSection list={secrets} selected={selected} onSelect={handleSelect} />
      )}
      {certificates != null && (
        <CertificateSection
          list={certificates}
          selected={selected}
          onSelect={handleSelect}
        />
      )}
    </Flex.Box>
  );
};

/** Lists the secrets and certificates the user can retrieve, in headed sections. */
export const Lists = ({ emptyContent }: ListsProps): ReactElement => {
  const lists = useLists();
  return <Body {...lists} emptyContent={emptyContent} />;
};
