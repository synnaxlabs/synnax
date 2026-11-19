// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/platform/settings/UsedBy.css";

import { type resource } from "@synnaxlabs/client";
import { Button } from "@synnaxlabs/lyra/button";
import { Dialog } from "@synnaxlabs/lyra/dialog";
import { Flex } from "@synnaxlabs/lyra/flex";
import { Text } from "@synnaxlabs/lyra/text";
import { stopPropagation } from "@synnaxlabs/lyra/util";
import { Resource } from "@synnaxlabs/pluto";
import { compare, location, state } from "@synnaxlabs/x";
import { type ReactElement, Suspense, useEffect, useRef, useState } from "react";

import { CSS } from "@/platform/css";
import { Tree } from "@/platform/tree";

/** Selects a use of a resource type by its key, keyed by the type. */
type Selects = Partial<Record<resource.Type, (key: string) => void>>;

interface UseProps {
  resource: resource.Resource;
  select?: (key: string) => void;
  close: () => void;
}

/** A resource that uses an item, shown with the icon of its type. */
const Use = ({ resource, select, close }: UseProps): ReactElement => {
  const Item = Tree.useItems()[resource.id.type] ?? Tree.DefaultItem;
  const renderButton = (onClick: () => void) => (
    <Button.Button
      variant="text"
      size="small"
      justify="start"
      full="x"
      className={CSS.BE("settings-used-by", "resource")}
      onClick={onClick}
    >
      {Item.icon(resource)}
      <Text.Text level="small" overflow="fade" color={10}>
        {resource.name}
      </Text.Text>
    </Button.Button>
  );
  if (select != null)
    return renderButton(() => {
      close();
      select(resource.id.key);
    });
  return (
    <Item.OnSelect>
      {(onSelect) => renderButton(() => onSelect(resource))}
    </Item.OnSelect>
  );
};

interface UnusedProps {
  kind: string;
}

const Unused = ({ kind }: UnusedProps): ReactElement => (
  <Text.Text level="small" color={9} className={CSS.BE("settings-used-by", "unused")}>
    Nothing uses this {kind}.
  </Text.Text>
);

/** Orders uses by name, then by key, so equal names keep one order. */
const USE_ORDER = compare.chain<resource.Resource>(
  compare.by((r) => r.name),
  compare.by((r) => r.id.key),
);

interface UsesProps extends UnusedProps, Pick<UseProps, "close"> {
  ids: resource.ID[];
  selects?: Selects;
}

const Uses = ({ ids, kind, selects, close }: UsesProps): ReactElement => {
  // A resource in the uses can be deleted between the read of the uses and this read.
  const resources = Resource.useMultiple({ ids });
  if (resources.length === 0) return <Unused kind={kind} />;
  return (
    <Flex.Box y empty className={CSS.BE("settings-used-by", "resources")}>
      {resources.toSorted(USE_ORDER).map((r) => (
        <Use
          key={`${r.id.type}:${r.id.key}`}
          resource={r}
          select={selects?.[r.id.type]}
          close={close}
        />
      ))}
    </Flex.Box>
  );
};

const POPOVER_LOCATION = {
  targetCorner: location.BOTTOM_RIGHT,
  dialogCorner: location.TOP_RIGHT,
};

// How long the popover stays open after the pointer leaves, so the pointer can cross
// the gap from the count to the popover.
const HOVER_CLOSE_DELAY = 150;

/** Returns the ID of the popover that lists the uses of the item with the key. */
export const usedByDialogID = (key: string): string => `used-by-${key}`;

type PopoverMode = "hover" | "pinned" | null;

export interface UsedByProps {
  /** The key of the item, which names its popover for {@link usedByDialogID}. */
  itemKey: string;
  /** The kind of the item, such as "secret". */
  kind: string;
  /** The resources that use the item, or undefined while they load. */
  uses?: resource.ID[];
  /** The element the popover opens below. */
  anchor: HTMLElement | null;
  /**
   * Selects a use of a resource type in place of its tree item, and closes the popover.
   * Keyed by the type.
   */
  selects?: Selects;
}

/**
 * The count of the resources that use an item. Hovering or clicking it opens a popover
 * that lists them, as does {@link usedByDialogID} through `Dialog.open`.
 */
export const UsedBy = ({
  itemKey,
  kind,
  uses,
  anchor,
  selects,
}: UsedByProps): ReactElement => {
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
    setMode((m) => (state.resolve(arg, m != null) ? "pinned" : null));
  const count = uses?.length ?? 0;
  return (
    <Dialog.Frame
      id={usedByDialogID(itemKey)}
      anchor={anchor}
      visible={mode != null}
      onVisibleChange={handleVisibleChange}
      location={POPOVER_LOCATION}
    >
      {count > 0 && (
        <Button.Button
          variant="text"
          size="small"
          textColor={9}
          onDoubleClick={stopPropagation}
          onPointerEnter={handleEnter}
          onPointerLeave={scheduleClose}
          onClick={() => setMode("pinned")}
        >
          Used by {count}
        </Button.Button>
      )}
      <Dialog.Dialog
        background={1}
        rounded="medium"
        className={CSS.BE("settings-used-by", "popover")}
        onPointerEnter={cancelClose}
        onPointerLeave={mode === "hover" ? scheduleClose : undefined}
        onClick={stopPropagation}
      >
        <Text.Text
          level="small"
          color={9}
          className={CSS.BE("settings-used-by", "label")}
        >
          Used by
        </Text.Text>
        {uses != null &&
          (uses.length === 0 ? (
            <Unused kind={kind} />
          ) : (
            <Suspense fallback={null}>
              <Uses
                ids={uses}
                kind={kind}
                selects={selects}
                close={() => setMode(null)}
              />
            </Suspense>
          ))}
      </Dialog.Dialog>
    </Dialog.Frame>
  );
};
