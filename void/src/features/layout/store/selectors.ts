import memoize from "proxy-memoize";
import { useCallback } from "react";
import { useSelector } from "react-redux";
import { LayoutContent } from "../types";
import { LayoutStoreState } from "./slice";

export const useSelectLayoutContent = <S, P>(key: string) =>
  useSelector(
    useCallback(
      memoize(
        (state: LayoutStoreState) =>
          state.layout.contents[key] as LayoutContent<S, P>
      ),
      [key]
    )
  );

export const useSelectMosaic = () =>
  useSelector(
    useCallback(
      memoize((state: LayoutStoreState) => state.layout.mosaic),
      []
    )
  );

export const useSelectWindowPlacement = (winKey: string) =>
  useSelector(
    useCallback(
      memoize((state: LayoutStoreState) => state.layout.placements[winKey]),
      [winKey]
    )
  );

export const useSelectLayoutRendererProps = <S, P>(key: string) =>
  useSelector(
    useCallback(
      memoize((state: LayoutStoreState) => {
        const placement = state.layout.placements[key];
        if (!placement) return undefined;
        const content = state.layout.contents[placement.contentKey];
        if (!content) return undefined;
        return {
          ...content,
          ...placement,
        };
      }),
      [key]
    )
  );
