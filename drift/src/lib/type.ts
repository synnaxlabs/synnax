export const DRIFT_NAME = 'drift';
export const DRIFT_ACTION_INDICATOR = 'DA';
export const DRIFT_PREFIX_SPLITTER = '<>';
export const DRIFT_KEY_SPLITTER = '@';

export const sugarType = (type: string, key: string) =>
  DRIFT_ACTION_INDICATOR.concat(
    DRIFT_KEY_SPLITTER,
    key,
    DRIFT_PREFIX_SPLITTER,
    type
  );

export const parseKey = (prefix?: string) => {
  if (!prefix) return '';
  const [indicator, winKey] = prefix.split(DRIFT_KEY_SPLITTER);
  return indicator === DRIFT_ACTION_INDICATOR && winKey ? winKey : null;
};

export const desugarType = (sugaredType: string) => {
  const [prefix, ...rest] = sugaredType.split(DRIFT_PREFIX_SPLITTER);
  const type = rest[rest.length - 1];
  if (prefix.includes(DRIFT_ACTION_INDICATOR)) {
    return { fromListener: true, type: type, key: parseKey(prefix) };
  }
  return { fromListener: false, type: sugaredType, key: '' };
};
