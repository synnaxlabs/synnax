// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { arc } from "@synnaxlabs/client";
import { Access, Arc as Base, Icon, Input } from "@synnaxlabs/pluto";
import { deep, uuid } from "@synnaxlabs/x";

import { useSelectVersion } from "@/arc/selectors";
import { internalCreate, type State } from "@/arc/slice";
import { ZERO_STATE } from "@/arc/types";
import { translateGraphToConsole } from "@/arc/types/translate";
import { TYPE } from "@/arc/types/v0";
import { createLoadRemote } from "@/hooks/useLoadRemote";
import { type Layout } from "@/layout";
import { Modals } from "@/modals";
import { type Selector } from "@/selector";

export const LAYOUT_TYPE = "arc_editor";
export type LayoutType = typeof LAYOUT_TYPE;

const ARC_CREATE_LAYOUT_TYPE = "arc_create";

const [useCreate, Create] = Modals.createBase<string, Modals.BaseArgs<string>>(
  "Arc", ARC_CREATE_LAYOUT_TYPE, ({ value: { result }, onFinish }) => <div>
    <Input.Text value={result} onChange={onFinish} />
  </div>);

export const SELECTABLE: Selector.Selectable<string, Modals.BaseArgs<string>> = {
  key: LAYOUT_TYPE,
  title: "Arc Automation",
  icon: <Icon.Arc />,
  useVisible: () => Access.useUpdateGranted(arc.TYPE_ONTOLOGY_ID),
  create: async ({ layoutKey, createModal }) => {
    const name = await createModal({});
    if (name == null) return null;
    return create({ key: layoutKey, name });
  },
  useModal: Modals.useRename(),
};

export type CreateArg = Partial<State> & Partial<Layout.BaseState>;

export const create =
  (initial: CreateArg = {}): Layout.Creator =>
  ({ dispatch }) => {e
    const { name = "Arc Editor", location = "mosaic", tab, mode, ...rest } = initial;
    const key = arc.keyZ.safeParse(initial.key).data ?? uuid.create();
    dispatch(internalCreate({ ...deep.copy(ZERO_STATE), ...rest, key, mode }));
    return {
      key,
      location,
      name,
      icon: "Arc",
      type: LAYOUT_TYPE,
      window: { navTop: true, showTitle: true },
      tab,
    };
  };

export const useLoadRemote = createLoadRemote<arc.Arc>({
  useRetrieve: Base.useRetrieveObservable,
  targetVersion: ZERO_STATE.version,
  useSelectVersion,
  actionCreator: (v) =>
    internalCreate({
      version: "0.0.0",
      key: v.key,
      type: TYPE,
      remoteCreated: true,
      graph: translateGraphToConsole(v.graph),
      text: v.text,
      mode: v.mode,
    }),
});
