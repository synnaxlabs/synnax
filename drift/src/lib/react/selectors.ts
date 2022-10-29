import memoize from 'proxy-memoize';
import { useCallback } from 'react';
import { useSelector } from 'react-redux';

import { StoreState } from '../slice';

export const useSelectWindowStatus = (key?: string) =>
  useSelector(
    useCallback(
      memoize((state: StoreState) => {
        return state.drift.windows[key || state.drift.key].status;
      }),
      [key]
    )
  );

export const useSelectWindow = (key?: string) =>
  useSelector(
    useCallback(
      memoize(
        (state: StoreState) => state.drift.windows[key || state.drift.key]
      ),
      [key]
    )
  );
