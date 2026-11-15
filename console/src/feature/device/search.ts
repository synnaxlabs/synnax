// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { device, group, rack, resource } from "@synnaxlabs/client";
import { Device, Group as PGroup, Rack } from "@synnaxlabs/pluto";
import { unique } from "@synnaxlabs/x";

import { Group } from "@/platform/group";
import { Tree } from "@/platform/tree";

// A device sits in its group, else in the device that holds it, else on its rack.
const deviceParent = (d: device.Device): resource.ID => {
  if (d.group != null) return group.resourceID(d.group);
  if (d.parent != null) return device.resourceID(d.parent);
  return rack.resourceID(d.rack);
};

const rackParent = (r: rack.Rack, root: resource.ID): resource.ID =>
  r.group == null ? root : group.resourceID(r.group);

const groupParent = (g: group.Group): resource.ID =>
  g.parent ?? { type: g.scope, key: "" };

const toDeviceHit = (d: device.Device): Tree.Hit => ({
  ...Tree.toResource("device")(d),
  parent: deviceParent(d),
});

const useMatches: Tree.UseMatches = (query) => {
  const params = query == null ? null : { searchTerm: query.term, limit: query.limit };
  const groups = Group.useMatches(query);
  const racks = Rack.useResultMultiple(params);
  const devices = Device.useResultMultiple(params);
  if (query == null) return [{ variant: "loading" }];
  const { root } = query;
  return [
    ...groups,
    Tree.toMatches(racks, (r) => ({
      ...Tree.toResource("rack")(r.payload),
      parent: rackParent(r, root),
    })),
    Tree.toMatches(devices, toDeviceHit),
  ];
};

/** A node between a match and the root, and the node that holds it. */
interface Folder {
  resource: Tree.Resource;
  parent: resource.ID;
}

const place = (hits: Tree.Hit[], folders: Folder[], root: resource.ID) => {
  const byKey = new Map(folders.map((f) => [f.resource.key, f]));
  const used = new Map<string, Tree.Resource>();
  const paths: Record<string, resource.ID[][]> = {};
  hits.forEach((hit) => {
    const path: Tree.Resource[] = [];
    let { parent } = hit;
    while (!resource.idsEqual(parent, root)) {
      const folder = byKey.get(resource.idToString(parent));
      // A match whose folders lead out of the root does not show.
      if (folder == null) return;
      path.unshift(folder.resource);
      parent = folder.parent;
    }
    path.forEach((r) => used.set(r.key, r));
    paths[hit.key] = [path.map(({ id }) => id)];
  });
  return { resources: [...used.values()], paths };
};

// A device group sits under a rack, and a rack can sit in a group, so the folders
// above a match can alternate between the two. Every rack and group is fetched at
// once, because a cluster has few of them.
const usePlacement: Tree.UsePlacement = (query) => {
  const asked = query != null && query.matches.length > 0;
  const chassisKeys = unique.unique(
    query?.matches.flatMap(({ parent }) =>
      parent.type === "device" ? [parent.key] : [],
    ) ?? [],
  );
  const chassisResult = Device.useResultAncestors(
    chassisKeys.length === 0 ? null : { keys: chassisKeys },
  );
  const rackResult = Rack.useResultMultiple(asked ? {} : null);
  const groupResult = PGroup.useResultList(asked ? { scope: query.root.type } : null);
  if (query == null) return { variant: "loading" };
  const { matches, root } = query;
  if (!asked) return { variant: "success", placement: place(matches, [], root) };
  for (const result of [chassisResult, rackResult, groupResult])
    if (result.variant === "error")
      return { variant: "error", description: result.status.description };
  const chassis =
    chassisKeys.length === 0
      ? []
      : chassisResult.variant === "success"
        ? chassisResult.data
        : null;
  if (
    chassis == null ||
    rackResult.variant !== "success" ||
    groupResult.variant !== "success"
  )
    return { variant: "loading" };
  const racks = rackResult.data;
  const groups = groupResult.data;
  const folders: Folder[] = [
    ...chassis.map((d) => ({
      resource: Tree.toResource("device")(d),
      parent: deviceParent(d),
    })),
    ...racks.map((r) => ({
      resource: Tree.toResource("rack")(r.payload),
      parent: rackParent(r, root),
    })),
    ...groups.map((g) => ({
      resource: Tree.toResource("group")(g),
      parent: groupParent(g),
    })),
  ];
  return { variant: "success", placement: place(matches, folders, root) };
};

/** Searches the groups, racks, and devices of the device tree. */
export const useSearch = Tree.createSearch({
  noun: "devices",
  useMatches,
  usePlacement,
});
