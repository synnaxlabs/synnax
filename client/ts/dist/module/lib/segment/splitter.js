import { ValidationError } from '../errors';
import TypedSegment from './typed';
export default class Splitter {
    threshold;
    constructor(threshold) {
        this.threshold = threshold;
    }
    split(segment) {
        if (segment.size.smallerThan(this.threshold))
            return [segment];
        if (!segment.channel.density)
            throw new ValidationError('Cannot split segment because channel density is undefined');
        const splitPoint = this.threshold.valueOf() -
            (this.threshold.valueOf() % segment.channel.density.valueOf());
        const truncated = new TypedSegment(segment.channel, {
            ...segment.payload,
            data: segment.payload.data.slice(0, splitPoint),
        });
        const next = new TypedSegment(segment.channel, {
            ...segment.payload,
            data: segment.payload.data.slice(splitPoint),
        });
        return [truncated, ...this.split(next)];
    }
}
//# sourceMappingURL=data:application/json;base64,eyJ2ZXJzaW9uIjozLCJmaWxlIjoic3BsaXR0ZXIuanMiLCJzb3VyY2VSb290IjoiIiwic291cmNlcyI6WyIuLi8uLi8uLi8uLi9zcmMvbGliL3NlZ21lbnQvc3BsaXR0ZXIudHMiXSwibmFtZXMiOltdLCJtYXBwaW5ncyI6IkFBQUEsT0FBTyxFQUFFLGVBQWUsRUFBRSxNQUFNLFdBQVcsQ0FBQztBQUc1QyxPQUFPLFlBQVksTUFBTSxTQUFTLENBQUM7QUFFbkMsTUFBTSxDQUFDLE9BQU8sT0FBTyxRQUFRO0lBQzNCLFNBQVMsQ0FBTztJQUVoQixZQUFZLFNBQWU7UUFDekIsSUFBSSxDQUFDLFNBQVMsR0FBRyxTQUFTLENBQUM7SUFDN0IsQ0FBQztJQUVELEtBQUssQ0FBQyxPQUFxQjtRQUN6QixJQUFJLE9BQU8sQ0FBQyxJQUFJLENBQUMsV0FBVyxDQUFDLElBQUksQ0FBQyxTQUFTLENBQUM7WUFBRSxPQUFPLENBQUMsT0FBTyxDQUFDLENBQUM7UUFDL0QsSUFBSSxDQUFDLE9BQU8sQ0FBQyxPQUFPLENBQUMsT0FBTztZQUMxQixNQUFNLElBQUksZUFBZSxDQUN2QiwyREFBMkQsQ0FDNUQsQ0FBQztRQUNKLE1BQU0sVUFBVSxHQUNkLElBQUksQ0FBQyxTQUFTLENBQUMsT0FBTyxFQUFFO1lBQ3hCLENBQUMsSUFBSSxDQUFDLFNBQVMsQ0FBQyxPQUFPLEVBQUUsR0FBRyxPQUFPLENBQUMsT0FBTyxDQUFDLE9BQU8sQ0FBQyxPQUFPLEVBQUUsQ0FBQyxDQUFDO1FBQ2pFLE1BQU0sU0FBUyxHQUFHLElBQUksWUFBWSxDQUFDLE9BQU8sQ0FBQyxPQUFPLEVBQUU7WUFDbEQsR0FBRyxPQUFPLENBQUMsT0FBTztZQUNsQixJQUFJLEVBQUUsT0FBTyxDQUFDLE9BQU8sQ0FBQyxJQUFJLENBQUMsS0FBSyxDQUFDLENBQUMsRUFBRSxVQUFVLENBQUM7U0FDaEQsQ0FBQyxDQUFDO1FBQ0gsTUFBTSxJQUFJLEdBQUcsSUFBSSxZQUFZLENBQUMsT0FBTyxDQUFDLE9BQU8sRUFBRTtZQUM3QyxHQUFHLE9BQU8sQ0FBQyxPQUFPO1lBQ2xCLElBQUksRUFBRSxPQUFPLENBQUMsT0FBTyxDQUFDLElBQUksQ0FBQyxLQUFLLENBQUMsVUFBVSxDQUFDO1NBQzdDLENBQUMsQ0FBQztRQUNILE9BQU8sQ0FBQyxTQUFTLEVBQUUsR0FBRyxJQUFJLENBQUMsS0FBSyxDQUFDLElBQUksQ0FBQyxDQUFDLENBQUM7SUFDMUMsQ0FBQztDQUNGIn0=