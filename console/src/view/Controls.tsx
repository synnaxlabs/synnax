// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/view/View.css";

import {
  Button,
  CSS as PCSS,
  Flex,
  Icon,
  Status,
  View as PView,
} from "@synnaxlabs/pluto";
import { location } from "@synnaxlabs/x";
import { type ReactElement, useCallback } from "react";

import { CSS } from "@/css";
import { Modals } from "@/modals";
import { useContext } from "@/view/context";
import { type Query } from "@/view/useQuery";

export interface ControlsProps<Q extends Query> {
  onCreate: () => void;
  query: Q;
}

export const Controls = <Q extends Query>({
  onCreate,
  query,
}: ControlsProps<Q>): ReactElement => {
  const { visible, editable, setEditable, resourceType } = useContext("View.Controls");
  const { update: create } = PView.useCreate();
  const handleError = Status.useErrorHandler();
  const renameModal = Modals.useRename();

  const handleEditableClick = useCallback(() => setEditable((e) => !e), [setEditable]);

  const handleCreateView = useCallback(() => {
    handleError(async () => {
      const name = await renameModal(
        { initialValue: `View for ${resourceType}` },
        { icon: "Status", name: "View.Create" },
      );
      if (name == null) return;
      create({ name, type: resourceType, query });
    }, "Failed to create view");
  }, [create, query, resourceType, renameModal, handleError]);

  return (
    <Flex.Box x className={CSS(CSS.BE("view", "buttons"), PCSS.visible(visible))} pack>
      <Button.Button
        onClick={onCreate}
        tooltipLocation={location.BOTTOM_LEFT}
        tooltip={`Create a ${resourceType}`}
      >
        <Icon.Add />
      </Button.Button>
      <Button.Toggle
        value={editable}
        onChange={handleEditableClick}
        tooltipLocation={location.BOTTOM_LEFT}
        tooltip={`${editable ? "Disable" : "Enable"} editing`}
      >
        {editable ? <Icon.EditOff /> : <Icon.Edit />}
      </Button.Toggle>
      <Button.Button
        onClick={handleCreateView}
        tooltipLocation={location.BOTTOM_LEFT}
        tooltip="Create a view"
      >
        <Icon.View />
      </Button.Button>
    </Flex.Box>
  );
};
