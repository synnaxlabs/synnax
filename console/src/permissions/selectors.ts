// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { access, type ontology, policy } from "@synnaxlabs/client";

import { useMemoSelect } from "@/hooks";
import { SLICE_NAME, type SliceState, type StoreState } from "@/permissions/slice";

const selectState = (state: StoreState): SliceState => state[SLICE_NAME];

export const selectPolicies = (state: StoreState): policy.Policy[] =>
  selectState(state).policies;

export const useSelectPolicies = (): policy.Policy[] =>
  useMemoSelect(selectPolicies, []);

export const selectCanUseType = (
  state: StoreState,
  type: ontology.ResourceType,
): boolean => {
  const policies = selectPolicies(state);
  return policies.some((p) =>
    p.objects.some((object) => {
      const oType = object.type;
      return (
        oType === policy.ALLOW_ALL_ONTOLOGY_TYPE ||
        (oType === type && p.actions.includes(access.ALL_ACTION))
      );
    }),
  );
};

export const useSelectCanUseType = (type: ontology.ResourceType): boolean =>
  useMemoSelect((state: StoreState) => selectCanUseType(state, type), [type]);

export const selectCanEditPolicies = (state: StoreState): boolean =>
  selectCanUseType(state, policy.ONTOLOGY_TYPE);

export const useSelectCanEditPolicies = (): boolean =>
  useMemoSelect(selectCanEditPolicies, []);
