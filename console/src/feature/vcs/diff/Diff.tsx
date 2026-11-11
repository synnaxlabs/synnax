// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/feature/vcs/diff/Diff.css";

import { resource, type vcs } from "@synnaxlabs/client";
import { Flex } from "@synnaxlabs/lyra/flex";
import { Status } from "@synnaxlabs/lyra/status";
import { Arc } from "@synnaxlabs/pluto";
import { type ReactElement, useEffect, useRef, useState } from "react";

import { Change } from "@/feature/vcs/diff/change";
import { CodeDiff } from "@/feature/vcs/diff/CodeDiff";
import { Header } from "@/feature/vcs/diff/Header";
import { PropertyList } from "@/feature/vcs/diff/PropertyList";
import { type Opened, type Target, useOpened } from "@/feature/vcs/diff/Provider";
import {
  type Files,
  parseFile,
  useFiles,
  useRefreshFiles,
} from "@/feature/vcs/diff/queries";
import { decodeSchematic, SchematicDiff } from "@/feature/vcs/diff/SchematicDiff";
import { CSS } from "@/platform/css";
import { Empty } from "@/platform/empty";
import { type Panel } from "@/platform/panel";
import { Toolbar } from "@/platform/toolbar";
import { Vcs } from "@/platform/vcs";

const kindOf = ({ before, after }: Files): vcs.ChangeKind | null => {
  if (before.length === 0 && after.length === 0) return null;
  if (before.length === 0) return "added";
  if (after.length === 0) return "deleted";
  return "modified";
};

const sameFiles = (before: vcs.File[], after: vcs.File[]): boolean =>
  before.length === after.length &&
  before.every(({ path, content }) =>
    after.some((file) => file.path === path && file.content === content),
  );

const OMIT = ["key"];

const HASH = /^[0-9a-f]{7,40}$/;

/**
 * Returns a key that changes whenever the files of the target can change, or null when
 * both revs name fixed commits.
 */
const useLiveKey = ({ resource, before, after }: Target): string | null => {
  const { status } = Vcs.useStatus();
  if (HASH.test(before) && HASH.test(after)) return null;
  const change = status?.changes.find((c) => c.resource === resource);
  return `${status?.head?.hash ?? ""}:${change?.digest ?? ""}`;
};

const propertiesOf = (
  kind: vcs.ChangeKind,
  files: Files,
  name: Change.Name,
): Change.Property[] => {
  const before = parseFile(files.before, ".json");
  const after = parseFile(files.after, ".json");
  const opts = { name, omit: OMIT };
  if (kind === "added") return Change.listProperties(after, "added", opts);
  if (kind === "deleted") return Change.listProperties(before, "removed", opts);
  return Change.diffProperties(before, after, opts);
};

interface PropertiesProps {
  properties: Change.Property[];
}

const Properties = ({ properties }: PropertiesProps): ReactElement => (
  <div className={CSS.BE("vcs-diff", "scroll")}>
    <PropertyList properties={properties} />
  </div>
);

interface BodyProps {
  target: Target;
  kind: vcs.ChangeKind;
  files: Files;
  json: boolean;
}

const Body = ({ target, kind, files, json }: BodyProps): ReactElement => {
  if (json)
    return (
      <CodeDiff
        target={target}
        kind={kind}
        before={files.before}
        after={files.after}
        extension=".json"
        language="json"
      />
    );
  const names = new Map(files.resources.map(({ key, name }) => [key, name]));
  const name: Change.Name = (id) => names.get(resource.idToString(id));
  const properties = propertiesOf(kind, files, name);
  const list =
    properties.length === 0 ? (
      <Empty.Action message="No property changes" grow />
    ) : (
      <Properties properties={properties} />
    );
  if (target.type === "schematic") {
    const before = decodeSchematic(files.before);
    const after = decodeSchematic(files.after);
    if ((kind !== "added" && before == null) || (kind !== "deleted" && after == null))
      return list;
    return (
      <SchematicDiff
        target={target}
        kind={kind}
        before={before}
        after={after}
        name={name}
      />
    );
  }
  const hasSource = [...files.before, ...files.after].some(({ path }) =>
    path.endsWith(".arc"),
  );
  if (target.type === "arc" && hasSource)
    return (
      <Flex.Box y empty grow>
        {properties.length > 0 && <Properties properties={properties} />}
        <CodeDiff
          target={target}
          kind={kind}
          before={files.before}
          after={files.after}
          extension=".arc"
          language={Arc.NAME}
        />
      </Flex.Box>
    );
  return list;
};

const Content = ({ target, version }: Opened): ReactElement => {
  const [json, setJson] = useState(false);
  const query = {
    resource: target.resource,
    before: target.before,
    after: target.after,
    version,
  };
  const result = useFiles(query);
  const refresh = useRefreshFiles();
  const live = useLiveKey(target);
  const lastLive = useRef(live);
  useEffect(() => {
    if (lastLive.current === live) return;
    lastLive.current = live;
    refresh(query);
  }, [live]);
  const files = result.data;
  const kind = files == null ? null : kindOf(files);
  const unchanged =
    files != null && kind != null && sameFiles(files.before, files.after);
  let body: ReactElement;
  if (result.variant === "error")
    body = (
      <Flex.Box center grow>
        <Status.Summary status={result.status} />
      </Flex.Box>
    );
  else if (files == null) body = <Status.Loading grow />;
  else if (kind == null)
    body = <Empty.Action message="This resource has no files at either version" grow />;
  else if (unchanged) body = <Empty.Action message="No changes" grow />;
  else
    body = (
      <Body
        key={`${target.resource}:${version}`}
        target={target}
        kind={kind}
        files={files}
        json={json}
      />
    );
  return (
    <Toolbar.Content className={CSS.B("vcs-diff")}>
      <Header
        target={target}
        kind={unchanged ? null : kind}
        json={json}
        onJsonChange={setJson}
      />
      <Toolbar.Body>{body}</Toolbar.Body>
    </Toolbar.Content>
  );
};

/** The diff tab: what one resource changed between two revs. */
export const Diff: Panel.Content = () => {
  const opened = useOpened();
  if (opened == null)
    return <Empty.Action message="Open a change to see what it changed" grow />;
  return <Content target={opened.target} version={opened.version} />;
};
