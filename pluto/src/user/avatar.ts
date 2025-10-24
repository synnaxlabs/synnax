const stringToHash = (str: string): number => {
  let hash = 0;
  for (let i = 0; i < str.length; i++) {
    hash = str.charCodeAt(i) + ((hash << 5) - hash);
    hash |= 0;
  }
  return Math.abs(hash);
};

const hashToHSL = (hash: number, offset: number = 0): string => {
  const hue = (hash + offset) % 360;
  const saturation = 60 + (hash % 30); // Range: 60–89%
  const lightness = 50 + (hash % 10); // Range: 50–59%
  return `hsl(${hue}, ${saturation}%, ${lightness}%)`;
};

export const avatar = (username: string): string => {
  const baseHash = stringToHash(username);
  const color1 = hashToHSL(baseHash, 0);
  const color2 = hashToHSL(baseHash, 120);
  const color3 = hashToHSL(baseHash, 240);
  return `linear-gradient(135deg, ${color1}, ${color2}, ${color3})`;
};
