import { useCallback } from "react";

import type { MosaicLeaf, Theme } from "@synnaxlabs/pluto";
import memoize from "proxy-memoize";
import { useSelector } from "react-redux";

import { Layout } from "../types";

import { LayoutStoreState } from "./slice";

export const selectLayout = (
  state: LayoutStoreState,
  key: string
): Layout | undefined => state.layout.layouts[key];

export const useSelectLayout = (key: string): Layout | undefined =>
  useSelector(
    useCallback(
      memoize((state: LayoutStoreState) => selectLayout(state, key)),
      [key]
    )
  );

export const useSelectMosaic = (): MosaicLeaf =>
  useSelector(
    useCallback(
      memoize((state: LayoutStoreState) => state.layout.mosaic),
      []
    )
  );

export const useSelectTheme = (): Theme => {
  const theme = useSelector(
    useCallback(
      memoize((state: LayoutStoreState) => state.layout.themes[state.layout.theme]),
      []
    )
  );
  return theme;
};
