import memoize from 'proxy-memoize';
import { useCallback } from 'react';
import { useSelector } from 'react-redux';

import { StoreState } from '../state';

/**
 * Selects the status of the window with the given key.
 *
 * @param key - The key of the window to select the status of.
 * If not provided, the status of the current window is selected.
 * @returns The status of the window.
 */
export const useSelectWindowStatus = (key?: string) =>
  useSelector(
    useCallback(
      memoize((state: StoreState) => {
        return state.drift.windows[key || state.drift.key].state;
      }),
      [key]
    )
  );

/**
 * Selects the window with the given key.
 *
 * @param key - The key of the window to select.
 * If not provided, the current window is selected.
 * @returns The window.
 */
export const useSelectWindow = (key?: string) =>
  useSelector(
    useCallback(
      memoize(
        (state: StoreState) => state.drift.windows[key || state.drift.key]
      ),
      [key]
    )
  );
