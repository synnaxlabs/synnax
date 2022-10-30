import { Size } from '../telem';
import TypedSegment from './typed';
export default class Splitter {
    threshold: Size;
    constructor(threshold: Size);
    split(segment: TypedSegment): TypedSegment[];
}
