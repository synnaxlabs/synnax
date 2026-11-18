// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/platform/file-browser/Modal.css";

import { type rack, status, TimeSpan, type TimeStamp } from "@synnaxlabs/client";
import { Button } from "@synnaxlabs/lyra/button";
import { Flex } from "@synnaxlabs/lyra/flex";
import { useAsyncEffect, useNow } from "@synnaxlabs/lyra/hooks";
import { Icon } from "@synnaxlabs/lyra/icon";
import { List } from "@synnaxlabs/lyra/list";
import { Modal as Base } from "@synnaxlabs/lyra/modal";
import { Nav } from "@synnaxlabs/lyra/nav";
import { Select } from "@synnaxlabs/lyra/select";
import { Status } from "@synnaxlabs/lyra/status";
import { Text } from "@synnaxlabs/lyra/text";
import { Triggers as PTriggers } from "@synnaxlabs/lyra/triggers";
import { Rack, Synnax } from "@synnaxlabs/pluto";
import { type ReactElement, useState } from "react";

import { CSS } from "@/platform/css";
import { type Entry, list, type Listing } from "@/platform/file-browser/list";
import { Modals } from "@/platform/modals";
import { Validity } from "@/platform/secret/validity";

const nextMinute = (now: TimeStamp): TimeStamp => now.add(TimeSpan.MINUTE);

interface KeyedEntry extends Entry {
  key: string;
}

const iconOf = (entry: Entry): Icon.FC => {
  if (entry.kind === "directory") return Icon.Folder;
  return entry.certificate == null ? Icon.File : Icon.Certificate;
};

interface EntryItemProps extends List.ItemProps<string> {
  now: TimeStamp;
  onPick: (entry: Entry) => void;
}

const EntryItem = ({ now, onPick, ...rest }: EntryItemProps): ReactElement | null => {
  const entry = List.useItem<string, KeyedEntry>(rest.itemKey);
  if (entry == null) return null;
  const EntryIcon = iconOf(entry);
  const { certificate } = entry;
  const validity =
    certificate == null ? null : Validity.describeDetails(certificate, now);
  const handleDoubleClick = () => {
    if (entry.kind === "file") onPick(entry);
  };
  return (
    <Select.Item
      {...rest}
      align="center"
      gap="medium"
      textValue={entry.name}
      onDoubleClick={handleDoubleClick}
    >
      <EntryIcon className={CSS.BE("file-browser", "icon")} />
      <Flex.Box y empty grow className={CSS.BE("file-browser", "text")}>
        <Text.Text overflow="fade">{entry.name}</Text.Text>
        {certificate != null && validity != null && (
          <Text.Text level="tiny" color={9} status={validity.status} overflow="fade">
            {`${certificate.subject} · ${validity.text}`}
          </Text.Text>
        )}
      </Flex.Box>
    </Select.Item>
  );
};

interface Target {
  path: string;
}

interface Loaded {
  target: Target;
  listing: Listing;
}

export interface ModalParams {
  /** The rack whose files the modal lists. */
  rack: rack.Key;
}

const Content = ({
  close,
  rack,
}: Modals.ContentProps<ModalParams, Entry>): ReactElement => {
  const client = Synnax.use();
  const rackResult = Rack.useResult({ key: rack });
  const now = useNow(nextMinute);
  // A new object for each request, so a request for the same folder lists it again.
  const [target, setTarget] = useState<Target>({ path: "" });
  const [loaded, setLoaded] = useState<Loaded | null>(null);
  const [failure, setFailure] = useState<status.Status | null>(null);
  const [picked, setPicked] = useState<string | null>(null);
  // Not a Flux query: its error holds no data, and a failed folder keeps the listing.
  useAsyncEffect(
    async (signal) => {
      if (client == null) return;
      setFailure(null);
      try {
        const listing = await list(client, rack, target.path);
        if (signal.aborted) return;
        setLoaded({ target, listing });
        setPicked(null);
      } catch (e) {
        if (signal.aborted) return;
        setFailure(status.fromException(e, "Failed to list files"));
      }
    },
    [client, rack, target],
  );
  const listing = loaded?.listing ?? null;
  const entries: KeyedEntry[] =
    listing?.entries.map((e) => ({ ...e, key: e.path })) ?? [];
  const { data, getItem } = List.useStaticData<string, KeyedEntry>({ data: entries });
  const handleChange = (key: string | null) => {
    if (key == null) return setPicked(null);
    const entry = getItem(key);
    if (entry == null) return;
    if (entry.kind === "directory") setTarget({ path: entry.path });
    else setPicked(key);
  };
  const pickedEntry = picked == null ? undefined : getItem(picked);
  const atTop = listing == null || listing.path === "";
  const loading = failure == null && loaded?.target !== target;
  const rackName = rackResult.variant === "success" ? rackResult.data.name : null;
  return (
    <Base.Frame>
      <Base.Header icon={<Icon.Folder />}>
        {rackName == null ? "Browse files" : ["Browse files", rackName]}
      </Base.Header>
      <Base.Body className={CSS.B("file-browser")}>
        <Flex.Box x align="center" gap="small">
          <Button.Button
            variant="text"
            size="small"
            textColor={9}
            tooltip="Up"
            disabled={atTop}
            onClick={() => setTarget({ path: listing?.parent ?? "" })}
          >
            <Icon.Arrow.Up />
          </Button.Button>
          <Text.Text level="small" variant="code" overflow="fade">
            {listing?.path ?? ""}
          </Text.Text>
        </Flex.Box>
        {failure != null && <Status.Summary level="small" status={failure} />}
        <Flex.Box y grow className={CSS.BE("file-browser", "entries")}>
          {loading && entries.length === 0 ? (
            <Text.Text level="small" color={8}>
              Listing files
            </Text.Text>
          ) : listing != null && entries.length === 0 ? (
            <Text.Text level="small" color={8}>
              The folder is empty
            </Text.Text>
          ) : (
            <Select.Frame<string, KeyedEntry>
              data={data}
              getItem={getItem}
              value={picked ?? undefined}
              onChange={handleChange}
            >
              <List.Scroll>
                <List.Items<string>>
                  {({ key, ...p }) => (
                    <EntryItem key={key} {...p} now={now} onPick={close} />
                  )}
                </List.Items>
              </List.Scroll>
            </Select.Frame>
          )}
        </Flex.Box>
        {listing?.truncated === true && (
          <Text.Text level="small" color={9}>
            {`Shows the first ${entries.length} entries.`}
          </Text.Text>
        )}
      </Base.Body>
      <Base.Footer>
        <Nav.Bar.End x align="center">
          <Button.Button
            variant="outlined"
            onClick={() => close()}
            triggerIndicator={PTriggers.ESCAPE}
          >
            Cancel
          </Button.Button>
          <Button.Button
            variant="filled"
            disabled={pickedEntry == null}
            onClick={() => {
              if (pickedEntry != null) close(pickedEntry);
            }}
          >
            Select
          </Button.Button>
        </Nav.Bar.End>
      </Base.Footer>
    </Base.Frame>
  );
};

/**
 * Opens a modal that browses the files of a rack. Resolves with the picked file, or
 * null on cancel.
 */
export const useModal = Modals.createPrompt<Entry, ModalParams>(Content);
