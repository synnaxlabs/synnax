export const DRIFT_NAME = 'drift';
export const DRIFT_ACTION_INDICATOR = 'DA';
export const DRIFT_PREFIX_SPLITTER = '<>';
export const DRIFT_WIN_KEY_SPLITTER = '@';

export const embedDriftMD = ({
  type,
  winKey: winID,
}: {
  type: string;
  winKey: string;
}) =>
  DRIFT_ACTION_INDICATOR.concat(
    DRIFT_WIN_KEY_SPLITTER,
    winID,
    DRIFT_PREFIX_SPLITTER,
    type
  );

export const parseWinKey = (prefix?: string) => {
  if (!prefix) return '';
  const [indicator, winKey] = prefix.split(DRIFT_WIN_KEY_SPLITTER);
  return indicator === DRIFT_ACTION_INDICATOR && winKey ? winKey : null;
};

export const parseDriftMD = (type: string) => {
  const [prefix, ...rest] = type.split(DRIFT_PREFIX_SPLITTER);
  const baseType = rest[rest.length - 1];
  if (prefix.includes(DRIFT_ACTION_INDICATOR)) {
    return { fromListener: true, baseType, winID: parseWinKey(prefix) };
  }
  return { fromListener: false, baseType: type, winKey: '' };
};
