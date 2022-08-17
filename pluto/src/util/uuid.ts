import { v4 as uuidv4 } from "uuid";

const uuidShortLength = 6;

export const uuidShort = () => uuidv4().substring(0, uuidShortLength);
