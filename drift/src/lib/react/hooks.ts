import { useEffect, useRef } from 'react';

import { useSelectWindowStatus } from './selectors';

export const useWindowLifecycle = (fn: () => () => void, key?: string) => {
  const status = useSelectWindowStatus(key);
  const onUnmount = useRef<undefined | (() => void)>(undefined);

  useEffect(() => {
    if (status === 'created' && !onUnmount.current) {
      onUnmount.current = fn();
    }
    if (status === 'closing' && onUnmount.current) {
      onUnmount.current();
      onUnmount.current = undefined;
    }
  }, [status, onUnmount]);
};
