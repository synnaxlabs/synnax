// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { schematic, type vcs } from "@synnaxlabs/client";
import { Flex } from "@synnaxlabs/lyra/flex";
import { Text } from "@synnaxlabs/lyra/text";
import { Control, Diagram, Schematic, Synnax, Viewport } from "@synnaxlabs/pluto";
import { caseconv, uuid, xy } from "@synnaxlabs/x";
import { type ReactElement, useEffect, useState } from "react";

import { Change } from "@/feature/vcs/diff/change";
import { PropertyList } from "@/feature/vcs/diff/PropertyList";
import { type Target } from "@/feature/vcs/diff/Provider";
import { parseFile } from "@/feature/vcs/diff/queries";
import { CSS } from "@/platform/css";
import { Vcs } from "@/platform/vcs";
import { Vis } from "@/platform/vis";
import { Session } from "@/session";

/** Returns the schematic a version holds, or null when its file does not decode. */
export const decodeSchematic = (files: vcs.File[]): schematic.Schematic | null => {
  const raw = parseFile(files, ".json");
  if (raw == null) return null;
  const parsed = schematic.schematicZ.safeParse(
    caseconv.snakeToCamel(raw, { schema: schematic.schematicZ }),
  );
  return parsed.success ? parsed.data : null;
};

const symbolName = (config: schematic.ElementConfig): string =>
  Schematic.Node.isConfig(config)
    ? Schematic.Node.resolveSpec(config.variant).name
    : Change.humanize(config.variant);

const INITIAL_VIEWPORT: Diagram.Viewport = { position: xy.ZERO, zoom: 1 };
const NO_SELECTION: string[] = [];
const noop = () => {};

interface RenderProps {
  kinds?: Map<string, Change.Kind>;
}

const Render = ({ kinds }: RenderProps): ReactElement => {
  const [viewport, setViewport] = useState(INITIAL_VIEWPORT);
  const [fitViewOnResize, setFitViewOnResize] = useState(true);
  const visible = Session.Panel.useSelectIsTabVisible();
  // Every symbol gets the class, so no click reaches a valve or a setpoint.
  const nodeClassName = (key: string): string => {
    const kind = kinds?.get(key);
    return CSS.cls(
      CSS.BE("vcs-diff", "symbol"),
      kind != null && CSS.BEM("vcs-diff", "symbol", kind),
    );
  };
  return (
    <Schematic.Schematic
      editable={false}
      onEditableChange={noop}
      enableTriggers={false}
      viewport={viewport}
      onViewportChange={setViewport}
      viewportMode="pan"
      onViewportModeChange={noop}
      triggers={Viewport.DEFAULT_TRIGGERS.pan}
      fitViewOnResize={fitViewOnResize}
      setFitViewOnResize={setFitViewOnResize}
      selected={NO_SELECTION}
      visible={visible}
      nodeClassName={nodeClassName}
    >
      <Vis.Controls x>
        <Diagram.Controls.FitView />
      </Vis.Controls>
    </Schematic.Schematic>
  );
};

interface CanvasProps extends RenderProps {
  document: schematic.Schematic;
}

/**
 * Renders a stored version of a schematic read-only. The version goes into the
 * client's cache under a new key for as long as the canvas is mounted. Its control
 * symbols show live state but never take control. Nothing reaches the Core.
 */
const Canvas = ({ document, kinds }: CanvasProps): ReactElement | null => {
  const client = Synnax.use();
  const [key, setKey] = useState<schematic.Key | null>(null);
  useEffect(() => {
    if (client == null) return;
    const previewKey = uuid.create();
    const release = client.schematics.preview({
      key: previewKey,
      name: document.name,
      snapshot: true,
      nodes: document.nodes,
      edges: document.edges,
      configs: document.configs,
    });
    setKey(previewKey);
    return () => {
      setKey(null);
      release();
    };
  }, [client, document]);
  if (key == null) return null;
  return (
    <Schematic.Suspended schematicKey={key}>
      <Control.Controller name={document.name} disabled>
        <Render kinds={kinds} />
      </Control.Controller>
    </Schematic.Suspended>
  );
};

const COUNT_ORDER: Change.Kind[] = ["added", "changed", "removed"];

const countKinds = (
  kinds: Map<string, Change.Kind> | undefined,
): string | undefined => {
  if (kinds == null || kinds.size === 0) return undefined;
  const counts = new Map<Change.Kind, number>();
  kinds.forEach((kind) => counts.set(kind, (counts.get(kind) ?? 0) + 1));
  return COUNT_ORDER.filter((kind) => counts.has(kind))
    .map((kind) => `${counts.get(kind)} ${kind}`)
    .join(", ");
};

interface PaneProps extends CanvasProps {
  label: string;
}

const Pane = ({ label, document, kinds }: PaneProps): ReactElement => {
  const detail = countKinds(kinds);
  return (
    <Flex.Box y empty grow className={CSS.BE("vcs-diff", "pane")}>
      <Flex.Box
        x
        align="center"
        justify="between"
        className={CSS.BE("vcs-diff", "pane-header")}
      >
        <Text.Text level="small" weight={500} color={10}>
          {label}
        </Text.Text>
        {detail != null && (
          <Text.Text level="small" color={8}>
            {detail}
          </Text.Text>
        )}
      </Flex.Box>
      <div className={CSS.BE("vcs-diff", "canvas")}>
        <div className={CSS.BE("vcs-diff", "surface")}>
          <Canvas document={document} kinds={kinds} />
        </div>
      </div>
    </Flex.Box>
  );
};

const MARKER_KINDS: Record<Change.Kind, vcs.ChangeKind> = {
  added: "added",
  changed: "modified",
  removed: "deleted",
};

const VARIANT_LABELS: Record<Change.Element["variant"], string> = {
  symbol: "Symbol",
  connection: "Connection",
};

interface ElementListProps {
  elements: Change.Element[];
  document: Change.Property[];
}

const ElementList = ({ elements, document }: ElementListProps): ReactElement => (
  <Flex.Box y empty className={CSS.BE("vcs-diff", "changes")}>
    {document.length > 0 && (
      <Flex.Box y empty className={CSS.BE("vcs-diff", "element")}>
        <Flex.Box x align="center" gap="small">
          <Vcs.KindMarker kind="modified" />
          <Text.Text level="small" weight={500} color={10}>
            Schematic
          </Text.Text>
        </Flex.Box>
        <PropertyList properties={document} />
      </Flex.Box>
    )}
    {elements.map((element) => (
      <Flex.Box
        key={`${element.variant}:${element.key}`}
        y
        empty
        className={CSS.BE("vcs-diff", "element")}
      >
        <Flex.Box x align="center" gap="small">
          <Vcs.KindMarker kind={MARKER_KINDS[element.kind]} />
          <Text.Text level="small" weight={500} color={10} overflow="fade">
            {element.label}
          </Text.Text>
          <Text.Text level="small" color={8}>
            {VARIANT_LABELS[element.variant]}
          </Text.Text>
        </Flex.Box>
        {element.properties.length > 0 && (
          <PropertyList properties={element.properties} />
        )}
      </Flex.Box>
    ))}
  </Flex.Box>
);

export interface SchematicDiffProps {
  target: Target;
  kind: vcs.ChangeKind;
  /** before is null for an added schematic. */
  before: schematic.Schematic | null;
  /** after is null for a deleted schematic. */
  after: schematic.Schematic | null;
  name: Change.Name;
}

/**
 * Shows two versions of a schematic side by side, outlines each symbol that changed,
 * and lists the changes below. An added or deleted schematic shows its one version.
 */
export const SchematicDiff = ({
  target,
  kind,
  before,
  after,
  name,
}: SchematicDiffProps): ReactElement => {
  if (before == null || after == null) {
    const document = before ?? after;
    const label = before == null ? target.afterLabel : target.beforeLabel;
    return (
      <Flex.Box x empty grow className={CSS.BE("vcs-diff", "panes")}>
        {document != null && <Pane label={label} document={document} />}
      </Flex.Box>
    );
  }
  const changes = Change.diffElements(before, after, { name, symbolName });
  const hasChanges = changes.elements.length > 0 || changes.document.length > 0;
  return (
    <Flex.Box y empty grow>
      <Flex.Box x empty grow className={CSS.BE("vcs-diff", "panes")}>
        <Pane label={target.beforeLabel} document={before} kinds={changes.before} />
        <Pane label={target.afterLabel} document={after} kinds={changes.after} />
      </Flex.Box>
      {kind === "modified" && hasChanges && (
        <ElementList elements={changes.elements} document={changes.document} />
      )}
    </Flex.Box>
  );
};
