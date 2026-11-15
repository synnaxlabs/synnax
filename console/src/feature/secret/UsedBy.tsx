// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type certificate, type resource, type secret } from "@synnaxlabs/client";
import { Button } from "@synnaxlabs/lyra/button";
import { Dialog } from "@synnaxlabs/lyra/dialog";
import { Flex } from "@synnaxlabs/lyra/flex";
import { Text } from "@synnaxlabs/lyra/text";
import { stopPropagation } from "@synnaxlabs/lyra/util";
import { Resource, Secret } from "@synnaxlabs/pluto";
import { location, state } from "@synnaxlabs/x";
import { type ReactElement, Suspense, useEffect, useRef, useState } from "react";

import { CSS } from "@/platform/css";
import { Tree } from "@/platform/tree";

/** Shows a certificate that uses a secret. */
export interface SelectCertificate {
  (key: certificate.Key): void;
}

const NO_USES: resource.ID[] = [];

/** Every secret's uses, read in one request that a deleted secret cannot fail. */
const ALL_USES: Secret.RetrieveMultipleQuery = { includeUses: true };

/**
 * Returns the IDs of the resources that use the secret and that the caller can read.
 * Suspends while the uses load.
 */
const useUses = (key: secret.Key): resource.ID[] =>
  Secret.useMultiple(ALL_USES).find((s) => s.key === key)?.uses ?? NO_USES;

/**
 * Returns the count of the resources that use the secret and that the caller can read,
 * or zero while the uses load or after their read fails. Never suspends or throws.
 */
const useUsesCount = (key: secret.Key): number =>
  Secret.useResultMultiple(ALL_USES).data?.find((s) => s.key === key)?.uses?.length ??
  0;

interface UseProps {
  resource: resource.Resource;
  onSelectCertificate?: SelectCertificate;
}

/** A resource that uses a secret, shown with the icon of its type. */
const Use = ({ resource, onSelectCertificate }: UseProps): ReactElement => {
  const Item = Tree.useItems()[resource.id.type] ?? Tree.DefaultItem;
  const renderButton = (onClick: () => void) => (
    <Button.Button
      variant="text"
      size="small"
      justify="start"
      full="x"
      className={CSS.BE("secret-settings", "resource")}
      onClick={onClick}
    >
      {Item.icon(resource)}
      <Text.Text level="small" overflow="fade" color={10}>
        {resource.name}
      </Text.Text>
    </Button.Button>
  );
  if (resource.id.type === "certificate" && onSelectCertificate != null)
    return renderButton(() => onSelectCertificate(resource.id.key));
  return (
    <Item.OnSelect>
      {(onSelect) => renderButton(() => onSelect(resource))}
    </Item.OnSelect>
  );
};

interface UsesProps extends Pick<UseProps, "onSelectCertificate"> {
  ids: resource.ID[];
}

const Unused = (): ReactElement => (
  <Text.Text level="small" color={9} className={CSS.BE("secret-settings", "unused")}>
    Nothing uses this secret.
  </Text.Text>
);

const Uses = ({ ids, onSelectCertificate }: UsesProps): ReactElement => {
  // A resource in the uses can be deleted between the read of the uses and this read.
  const resources = Resource.useMultiple({ ids });
  if (resources.length === 0) return <Unused />;
  return (
    <Flex.Box y empty className={CSS.BE("secret-settings", "resources")}>
      {resources.map((r) => (
        <Use
          key={`${r.id.type}:${r.id.key}`}
          resource={r}
          onSelectCertificate={onSelectCertificate}
        />
      ))}
    </Flex.Box>
  );
};

interface UsedByProps extends Pick<UseProps, "onSelectCertificate"> {
  secretKey: secret.Key;
}

/** Lists the resources that use a secret. Suspends while they load. */
const UsedBy = ({ secretKey, onSelectCertificate }: UsedByProps): ReactElement => {
  const uses = useUses(secretKey);
  if (uses.length === 0) return <Unused />;
  return <Uses ids={uses} onSelectCertificate={onSelectCertificate} />;
};

const POPOVER_LOCATION = {
  targetCorner: location.BOTTOM_RIGHT,
  dialogCorner: location.TOP_RIGHT,
};

// How long the popover stays open after the pointer leaves, so the pointer can cross
// the gap from the count to the popover.
const HOVER_CLOSE_DELAY = 150;

/** Returns the ID of the popover that lists the uses of a secret. */
export const usedByDialogID = (key: secret.Key): string => `used-by-${key}`;

type PopoverMode = "hover" | "pinned" | null;

interface CountProps extends Pick<
  Button.ButtonProps,
  "onPointerEnter" | "onPointerLeave" | "onClick"
> {
  secretKey: secret.Key;
}

const Count = ({ secretKey, ...rest }: CountProps): ReactElement | null => {
  const count = useUsesCount(secretKey);
  if (count === 0) return null;
  return (
    <Button.Button
      {...rest}
      variant="text"
      size="small"
      textColor={9}
      onDoubleClick={stopPropagation}
    >
      Used by {count}
    </Button.Button>
  );
};

export interface UsedByPopoverProps {
  secretKey: secret.Key;
  onSelectCertificate: SelectCertificate;
  /** The element the popover opens below. */
  anchor: HTMLElement | null;
}

/**
 * The count of the resources that use a secret. Hovering or clicking it opens a
 * popover that lists them, as does {@link usedByDialogID} through `Dialog.open`.
 */
export const UsedByPopover = ({
  secretKey,
  anchor,
  onSelectCertificate,
}: UsedByPopoverProps): ReactElement => {
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
      <Count
        secretKey={secretKey}
        onPointerEnter={handleEnter}
        onPointerLeave={scheduleClose}
        onClick={() => setMode("pinned")}
      />
      <Dialog.Dialog
        background={1}
        rounded="medium"
        className={CSS.BE("secret-settings", "popover")}
        onPointerEnter={cancelClose}
        onPointerLeave={mode === "hover" ? scheduleClose : undefined}
        onClick={stopPropagation}
      >
        <Text.Text
          level="small"
          color={9}
          className={CSS.BE("secret-settings", "label")}
        >
          Used by
        </Text.Text>
        <Suspense fallback={null}>
          <UsedBy secretKey={secretKey} onSelectCertificate={handleSelectCertificate} />
        </Suspense>
      </Dialog.Dialog>
    </Dialog.Frame>
  );
};
