import { access, ontology } from "@synnaxlabs/client";

import { useMemoSelect } from "@/hooks";
import { SLICE_NAME, type SliceState, type StoreState } from "@/policies/slice";

const selectState = (state: StoreState): SliceState => state[SLICE_NAME];

export const selectPolicies = (state: StoreState): access.Policy[] =>
  selectState(state).policies;

export const useSelectPolicies = (): access.Policy[] =>
  useMemoSelect(selectPolicies, []);

export const selectCanUseType = (
  state: StoreState,
  type: ontology.ResourceType,
): boolean => {
  const policies = selectPolicies(state);
  return policies.some(
    (policy) =>
      policy.objects.some((object) => new ontology.ID(object).matchesType(type)) &&
      policy.actions.includes(access.ALL_ACTION),
  );
};

export const useSelectCanUseType = (type: ontology.ResourceType): boolean =>
  useMemoSelect((state: StoreState) => selectCanUseType(state, type), [type]);
