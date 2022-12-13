import memoize from "proxy-memoize";
import { useCallback } from "react";
import { useSelector } from "react-redux";
import { WorkspaceStoreState } from "./slice";

export const useSelectRangeFilterCore = (state: WorkspaceStoreState, keys: string[]) =>
	state.workspace.ranges.filter((range) => keys.includes(range.key));

export const useSelectRanges = () => {
	return useSelector(
		useCallback(
			memoize((state: WorkspaceStoreState) => state.workspace.ranges),
			[]
		)
	);
};
