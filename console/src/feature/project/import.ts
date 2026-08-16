// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Store } from "@reduxjs/toolkit";
import {
  arc,
  DisconnectedError,
  group,
  lineplot,
  log,
  type ontology,
  panel,
  project,
  schematic,
  type Synnax,
  table,
  ValidationError,
} from "@synnaxlabs/client";
import { Access, type Status } from "@synnaxlabs/pluto";
import { errors, uuid } from "@synnaxlabs/x";
import { z } from "zod";

import { Import } from "@/platform/import";
import { Runtime } from "@/platform/runtime";
import { Session } from "@/session";

// NOTE: all client/console side project import code is temporary. Server-side project
// import replaces it before release.

/** The tiling file inside a legacy (layout-slice era) project export directory. */
export const LAYOUT_FILE_NAME = "LAYOUT.json";

/** The manifest file at the root of a project bundle written by the Core. */
export const MANIFEST_FILE_NAME = "manifest.json";

const legacyLayoutZ = z.object({
  key: z.string(),
  type: z.string(),
  name: z.string(),
});

// Legacy console-state exports carried the full layout slice; only the layout records
// are needed to locate and type each component file.
const legacySliceZ = z.object({
  layouts: z.record(z.string(), legacyLayoutZ),
});

type ComponentContext = Omit<Import.FileIngesterContext, "fileName">;

// Visualization layout types found in legacy (layout-slice era) project exports. Their
// component files are typeless legacy Console states, importable only through the
// server. Frozen — legacy exports are no longer produced.
const LEGACY_COMPONENT_TYPES = new Set<string>([
  arc.TYPE_ONTOLOGY_ID.type,
  lineplot.TYPE_ONTOLOGY_ID.type,
  log.TYPE_ONTOLOGY_ID.type,
  schematic.TYPE_ONTOLOGY_ID.type,
  table.TYPE_ONTOLOGY_ID.type,
]);

// Task layout types found in legacy project exports. Their component files are typed
// legacy task configs the Core migrates on import. Frozen for the same reason.
const LEGACY_TASK_TYPES = new Set<string>([
  "ethercat_read",
  "ethercat_write",
  "http_read",
  "http_write",
  "labjack_read",
  "labjack_write",
  "modbus_read",
  "modbus_write",
  "ni_analog_read",
  "ni_analog_write",
  "ni_counter_read",
  "ni_digital_read",
  "ni_digital_write",
  "opc_read",
  "opc_write",
  "pagerduty_alert",
]);

const ingestLegacy = async (
  legacyData: unknown,
  files: Import.File[],
  ctx: ComponentContext,
): Promise<void> => {
  const { layouts } = legacySliceZ.parse(legacyData);
  for (const [key, layout] of Object.entries(layouts)) {
    if (!LEGACY_COMPONENT_TYPES.has(layout.type) && !LEGACY_TASK_TYPES.has(layout.type))
      continue;
    const file = files.find(
      (file) =>
        file.name === `${layout.name}.json` ||
        file.name === `${key}.json` ||
        (typeof file.data === "object" &&
          file.data != null &&
          (("key" in file.data && file.data.key === key) ||
            ("name" in file.data && file.data.name === layout.name))),
    );
    if (file == null) throw new Error(`Data for ${key} not found`);
    await Import.ingestServer(file.data, { ...ctx, fileName: file.name });
  }
  // TODO(SY-4370): legacy exports carried a mosaic tiling for these layouts;
  // reconstructing it as panel documents is dropped, so a legacy import only
  // recreates the visualization documents.
};

const bundleManifestZ = z.object({
  version: z.number(),
  type: z.string(),
  name: z.string().optional(),
});

// A bundle panel envelope: the panel tree with each resource tab holding the target
// member's path from the bundle root instead of an ontology ID.
const bundlePanelZ = z.object({
  name: z.string().optional(),
  root: z.unknown(),
});

const pathOf = (file: { name: string; path?: string }): string =>
  file.path ?? file.name;

const parentDirOf = (path: string): string =>
  path.includes("/") ? path.slice(0, path.lastIndexOf("/")) : "";

const baseNameOf = (path: string): string => path.slice(path.lastIndexOf("/") + 1);

const trimJSONExtension = (name: string): string =>
  name.endsWith(".json") ? name.slice(0, -".json".length) : name;

// A member is every .json file in the bundle except the root manifest and the reserved
// legacy tiling file. Nested files keep their names: reservations apply at the root
// alone.
const isMember = (path: string): boolean =>
  path.endsWith(".json") && path !== MANIFEST_FILE_NAME && path !== LAYOUT_FILE_NAME;

// Rewrites each resource tab's bundle path to the ontology ID minted for that member,
// leaving every other field for panelZ to validate on create. Tabs referencing a
// skipped member are dropped.
const resolveNode = (
  node: unknown,
  refs: Map<string, ontology.ID>,
  skipped: Set<string>,
  panelName: string,
): unknown => {
  const n = z.record(z.string(), z.unknown()).parse(node);
  if (n.variant !== "leaf")
    return {
      ...n,
      first: resolveNode(n.first, refs, skipped, panelName),
      last: resolveNode(n.last, refs, skipped, panelName),
    };
  const tabs = z.array(z.record(z.string(), z.unknown())).parse(n.tabs ?? []);
  return {
    ...n,
    tabs: tabs.flatMap((tab) => {
      if (tab.variant !== "resource") return [tab];
      const path = z.string().parse(tab.resource);
      if (skipped.has(path)) return [];
      const id = refs.get(path);
      if (id == null)
        throw new Error(
          `Panel ${panelName} references ${path}, which is not in the bundle`,
        );
      return [{ ...tab, resource: id }];
    }),
  };
};

// TEMPORARY: client-side ingest of the Core-written bundle format, so an exported
// project round-trips until server-side project import replaces this whole file.
const ingestBundle = async (
  manifestData: unknown,
  directoryName: string,
  files: Import.File[],
  { client, store }: { client: Synnax; store: Store },
): Promise<void> => {
  const manifest = bundleManifestZ.parse(manifestData);
  if (manifest.type !== "project")
    throw new Error(`Cannot import a ${manifest.type} bundle as a project`);
  if (manifest.version !== 1)
    throw new Error(`Unsupported project bundle version ${manifest.version}`);
  const projectKey = uuid.create();
  await client.projects.create({
    key: projectKey,
    name: manifest.name || directoryName,
    layout: {},
  });
  const projectID = project.ontologyID(projectKey);
  // Each bundle directory becomes a group. Documents import under the project first,
  // because leaf importers require a project parent, then move into their group.
  const groupIDs = new Map<string, ontology.ID>([["", projectID]]);
  const groupFor = async (dir: string): Promise<ontology.ID> => {
    const existing = groupIDs.get(dir);
    if (existing != null) return existing;
    const parent = await groupFor(parentDirOf(dir));
    const created = await client.groups.create({ parent, name: baseNameOf(dir) });
    const id = group.ontologyID(created.key);
    groupIDs.set(dir, id);
    return id;
  };
  const members = files.filter((file) => isMember(pathOf(file)));
  const isPanelEnvelope = (data: unknown): boolean =>
    typeof data === "object" && data != null && "type" in data && data.type === "panel";
  const refs = new Map<string, ontology.ID>();
  // HACK: the Core exports members it cannot yet import back (tasks register only an
  // exporter), so a member rejected for a missing importer is skipped and every panel
  // tab referencing it is dropped. Server-side project import removes this.
  const skipped = new Set<string>();
  // Minted IDs accumulate per directory so each group receives one moveChildren call
  // instead of one per document.
  const moves = new Map<string, ontology.ID[]>();
  for (const member of members.filter(({ data }) => !isPanelEnvelope(data))) {
    const path = pathOf(member);
    let id: ontology.ID;
    try {
      id = await client.imex.import(JSON.stringify(member.data), {
        encoding: "JSON",
        fileName: baseNameOf(path),
        parent: projectID,
      });
    } catch (error) {
      if (
        !ValidationError.matches(error) ||
        !String(error).includes("no importer registered")
      )
        throw errors.fromUnknown(error);
      skipped.add(path);
      continue;
    }
    refs.set(path, id);
    const dir = parentDirOf(path);
    if (dir !== "") moves.set(dir, [...(moves.get(dir) ?? []), id]);
  }
  for (const [dir, ids] of moves)
    await client.ontology.moveChildren(projectID, await groupFor(dir), ...ids);
  for (const member of members.filter(({ data }) => isPanelEnvelope(data))) {
    const path = pathOf(member);
    const envelope = bundlePanelZ.parse(member.data);
    const name = envelope.name ?? trimJSONExtension(baseNameOf(path));
    await client.panels.create({
      key: uuid.create(),
      name,
      root: panel.nodeZ.parse(resolveNode(envelope.root, refs, skipped, name)),
      parent: await groupFor(parentDirOf(path)),
    });
  }
  store.dispatch(Session.Project.select(projectKey));
};

export const ingest: Import.DirectoryIngester = async (
  name,
  files,
  { client, store },
) => {
  if (!Access.updateGranted({ id: project.TYPE_ONTOLOGY_ID, client }))
    throw new Error("You do not have permission to import projects");
  if (client == null) throw new DisconnectedError();
  const manifestFile = files.find((file) => pathOf(file) === MANIFEST_FILE_NAME);
  if (manifestFile != null)
    return await ingestBundle(manifestFile.data, name, files, { client, store });
  const legacyFile = files.find((file) => file.name === LAYOUT_FILE_NAME);
  if (legacyFile == null) throw new Error(`${MANIFEST_FILE_NAME} not found`);
  const projectKey = uuid.create();
  // Create the project first so imported components can be parented to it.
  await client.projects.create({ key: projectKey, name, layout: {} });
  await ingestLegacy(legacyFile.data, files, { client, projectKey });
  store.dispatch(Session.Project.select(projectKey));
};

export interface IngestContext {
  handleError: Status.ErrorHandler;
  client: Synnax | null;
  store: Store;
}

export const import_ = ({ handleError, client, store }: IngestContext) => {
  let name: string | undefined = "project";
  handleError(async () => {
    const directory = await Runtime.pickDirectory({ title: "Import a Project" });
    if (directory == null) return;
    name = directory.name;
    const fileData = await Promise.all(
      directory.files
        .filter((file) => Import.isParsableFile(pathOf(file)))
        .map(async (file): Promise<Import.File> => ({
          name: file.name,
          path: file.path,
          data: JSON.parse(await file.read()),
        })),
    );
    await ingest(name, fileData, { client, store });
  }, `Failed to import ${name}`);
};
