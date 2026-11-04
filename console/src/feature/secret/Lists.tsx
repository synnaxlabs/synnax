// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { certificate, secret } from "@synnaxlabs/client";
import { Button } from "@synnaxlabs/lyra/button";
import { Dialog } from "@synnaxlabs/lyra/dialog";
import { Flex } from "@synnaxlabs/lyra/flex";
import { Icon } from "@synnaxlabs/lyra/icon";
import { List } from "@synnaxlabs/lyra/list";
import { Menu } from "@synnaxlabs/lyra/menu";
import { Select } from "@synnaxlabs/lyra/select";
import { Text } from "@synnaxlabs/lyra/text";
import { stopPropagation } from "@synnaxlabs/lyra/util";
import { Access, Certificate, Secret } from "@synnaxlabs/pluto";
import { location, state, TimeStamp } from "@synnaxlabs/x";
import { type ReactElement, type ReactNode, useEffect, useRef, useState } from "react";

import {
  CertificateMenu,
  FallbackMenu,
  SecretMenu,
} from "@/feature/secret/ContextMenu";
import {
  type SelectCertificate,
  UsedBy,
  UsesBoundary,
  useUses,
} from "@/feature/secret/UsedBy";
import { type Lists as ListsData, useLists } from "@/feature/secret/useLists";
import { describeCertificate, describeSecret } from "@/feature/secret/validity";
import { CSS } from "@/platform/css";

const POPOVER_LOCATION = {
  targetCorner: location.BOTTOM_RIGHT,
  dialogCorner: location.TOP_RIGHT,
};

// How long the popover stays open after the pointer leaves, so the pointer can cross
// the gap from the count to the popover.
const HOVER_CLOSE_DELAY = 150;

const usedByDialogID = (key: secret.Key): string => `used-by-${key}`;

type PopoverMode = "hover" | "pinned" | null;

interface UsedByPopoverProps {
  secretKey: secret.Key;
  anchor: HTMLElement | null;
  onSelectCertificate: SelectCertificate;
}

const UsedByPopover = ({
  secretKey,
  anchor,
  onSelectCertificate,
}: UsedByPopoverProps): ReactElement => {
  const count = useUses(secretKey).length;
  const [mode, setMode] = useState<PopoverMode>(null);
  const closeTimer = useRef<ReturnType<typeof setTimeout> | undefined>(undefined);
  useEffect(() => () => clearTimeout(closeTimer.current), []);
  const cancelClose = () => clearTimeout(closeTimer.current);
  const scheduleClose = () => {
    cancelClose();
    closeTimer.current = setTimeout(
      () => setMode((m) => (m === "hover" ? null : m)),
      HOVER_CLOSE_DELAY,
    );
  };
  const handleEnter = () => {
    cancelClose();
    setMode((m) => m ?? "hover");
  };
  const handleVisibleChange = (arg: state.SetArg<boolean>) =>
    setMode((m) => (state.executeSetter(arg, m != null) ? "pinned" : null));
  const handleSelectCertificate = (key: certificate.Key) => {
    setMode(null);
    onSelectCertificate(key);
  };
  return (
    <Dialog.Frame
      id={usedByDialogID(secretKey)}
      anchor={anchor}
      visible={mode != null}
      onVisibleChange={handleVisibleChange}
      location={POPOVER_LOCATION}
    >
      {count > 0 && (
        <Button.Button
          variant="text"
          size="tiny"
          textColor={9}
          className={CSS.BE("credentials", "count")}
          onPointerEnter={handleEnter}
          onPointerLeave={scheduleClose}
          onClick={() => setMode("pinned")}
          onDoubleClick={stopPropagation}
        >
          Used by {count}
        </Button.Button>
      )}
      <Dialog.Dialog
        background={1}
        rounded="medium"
        className={CSS.BE("credentials", "popover")}
        onPointerEnter={cancelClose}
        onPointerLeave={mode === "hover" ? scheduleClose : undefined}
        onClick={stopPropagation}
      >
        <Text.Text level="small" color={9} className={CSS.BE("credentials", "label")}>
          Used by
        </Text.Text>
        <UsesBoundary>
          <UsedBy secretKey={secretKey} onSelectCertificate={handleSelectCertificate} />
        </UsesBoundary>
      </Dialog.Dialog>
    </Dialog.Frame>
  );
};

interface SecretItemProps extends List.ItemProps<secret.Key> {
  onSelectCertificate: SelectCertificate;
}

const SecretItem = ({ onSelectCertificate, ...rest }: SecretItemProps) => {
  const { itemKey } = rest;
  const s = List.useItem<secret.Key, secret.Secret>(itemKey);
  const canUpdate = Access.useUpdateGranted(secret.ontologyID(itemKey));
  const { update: rename } = Secret.useRename();
  const [element, setElement] = useState<HTMLElement | null>(null);
  if (s == null) return null;
  return (
    <Select.Item
      {...rest}
      ref={setElement}
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
          weight={450}
        />
        <Text.Text level="tiny" color={9} overflow="fade">
          {describeSecret(s)}
        </Text.Text>
      </Flex.Box>
      <UsesBoundary>
        <UsedByPopover
          secretKey={itemKey}
          anchor={element}
          onSelectCertificate={onSelectCertificate}
        />
      </UsesBoundary>
    </Select.Item>
  );
};

interface CertificateItemProps extends List.ItemProps<certificate.Key> {
  now: TimeStamp;
}

const CertificateItem = ({ now, ...rest }: CertificateItemProps) => {
  const { itemKey } = rest;
  const c = List.useItem<certificate.Key, certificate.Certificate>(itemKey);
  const canUpdate = Access.useUpdateGranted(certificate.ontologyID(itemKey));
  const { update: rename } = Certificate.useRename();
  if (c == null) return null;
  const { text, status } = describeCertificate(c, now);
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
          weight={450}
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

export interface ListsProps {
  emptyContent: ReactNode;
}

interface BodyProps extends ListsProps, ListsData {}

const Body = ({ secrets, certificates, emptyContent }: BodyProps): ReactElement => {
  const [selected, setSelected] = useState<string | undefined>(undefined);
  const [now] = useState(() => TimeStamp.now());
  const secretMenu = Menu.useContextMenu();
  const certificateMenu = Menu.useContextMenu();
  const handleChange = (key: string | null) => setSelected(key ?? undefined);
  const selectCertificate = (key: certificate.Key) => setSelected(key);
  const openUsedBy = (key: secret.Key) => {
    setSelected(key);
    Dialog.open(usedByDialogID(key));
  };
  const renderSecretMenu = ({ keys }: Menu.ContextMenuMenuProps) => {
    const s = keys.length === 1 ? secrets.getItem(keys[0]) : undefined;
    if (s == null) return <FallbackMenu />;
    return <SecretMenu secret={s} onUsedBy={openUsedBy} onRename={renameByKey} />;
  };
  const renderCertificateMenu = ({ keys }: Menu.ContextMenuMenuProps) => {
    const c = keys.length === 1 ? certificates.getItem(keys[0]) : undefined;
    if (c == null) return <FallbackMenu />;
    return <CertificateMenu certificate={c} onRename={renameByKey} />;
  };
  if (
    secrets.answered &&
    certificates.answered &&
    secrets.data.length === 0 &&
    certificates.data.length === 0
  )
    return <>{emptyContent}</>;
  return (
    <Flex.Box y empty full className={CSS.BE("credentials", "lists")}>
      <Section title="Secrets">
        <Menu.ContextMenu menu={renderSecretMenu} {...secretMenu} />
        <Select.Frame<secret.Key, secret.Secret>
          data={secrets.data}
          getItem={secrets.getItem}
          subscribe={secrets.subscribe}
          value={selected}
          onChange={handleChange}
          allowNone
        >
          <List.Scroll
            onContextMenu={secretMenu.open}
            className={CSS.BE("credentials", "scroll")}
          >
            <List.Items<secret.Key, secret.Secret>
              emptyContent={secrets.answered && EMPTY_SECRETS}
            >
              {({ key, ...p }) => (
                <SecretItem key={key} {...p} onSelectCertificate={selectCertificate} />
              )}
            </List.Items>
          </List.Scroll>
        </Select.Frame>
      </Section>
      <Section title="Certificates">
        <Menu.ContextMenu menu={renderCertificateMenu} {...certificateMenu} />
        <Select.Frame<certificate.Key, certificate.Certificate>
          data={certificates.data}
          getItem={certificates.getItem}
          subscribe={certificates.subscribe}
          value={selected}
          onChange={handleChange}
          allowNone
        >
          <List.Scroll
            onContextMenu={certificateMenu.open}
            className={CSS.BE("credentials", "scroll")}
          >
            <List.Items<certificate.Key, certificate.Certificate>
              emptyContent={certificates.answered && EMPTY_CERTIFICATES}
            >
              {({ key, ...p }) => <CertificateItem key={key} {...p} now={now} />}
            </List.Items>
          </List.Scroll>
        </Select.Frame>
      </Section>
    </Flex.Box>
  );
};

/** Lists secrets and certificates in two headed sections. */
export const Lists = ({ emptyContent }: ListsProps): ReactElement => {
  const lists = useLists();
  return <Body {...lists} emptyContent={emptyContent} />;
};
