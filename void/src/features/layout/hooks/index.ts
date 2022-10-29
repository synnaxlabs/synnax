import { AnyAction } from "@reduxjs/toolkit";
import { closeWindow, createWindow } from "@synnaxlabs/drift";
import { Dispatch, useCallback } from "react";
import { useDispatch } from "react-redux";
import {
  placeLayout,
  removeLayout,
  setTheme,
  toggleTheme,
  useSelectLayout,
  useSelectTheme,
} from "../store";
import { Layout, LayoutWindowProps } from "../types";
import { ThemeProviderProps } from "@synnaxlabs/pluto";

export interface LayoutCreatorProps {
  dispatch: Dispatch<AnyAction>;
}

export type LayoutCreator = (props: LayoutCreatorProps) => Layout;

export type LayoutPlacer = (layout_: Layout | LayoutCreator) => void;

export const useLayoutPlacer = () => {
  const dispatch = useDispatch();
  return useCallback(
    (layout_: Layout | LayoutCreator) => {
      const layout =
        typeof layout_ === "function" ? layout_({ dispatch }) : layout_;
      const { key, location, window, title } = layout;
      dispatch(placeLayout(layout));
      if (location === "window")
        dispatch(
          createWindow({
            ...{ ...window, navTop: undefined },
            url: "/",
            key,
            title,
          })
        );
    },
    [dispatch]
  );
};

export const useLayoutRemover = (contentKey: string) => {
  const dispatch = useDispatch();
  const layout = useSelectLayout(contentKey);
  return () => {
    dispatch(removeLayout(contentKey));
    if (layout.location === "window")
      dispatch(closeWindow({ key: contentKey }));
  };
};

export const useThemeProvider = (): ThemeProviderProps => {
  const theme = useSelectTheme();
  const dispatch = useDispatch();
  return {
    theme,
    setTheme: (key: string) => dispatch(setTheme(key)),
    toggleTheme: () => dispatch(toggleTheme()),
  };
};
