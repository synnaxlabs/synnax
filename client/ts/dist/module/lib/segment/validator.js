import { ContiguityError, UnexpectedError, ValidationError } from '../errors';
export class ScalarTypeValidator {
    validate(array, dataType) {
        if (!dataType.checkArray(array)) {
            throw new ValidationError({
                field: 'data',
                message: `Data type mismatch. Expected ${dataType} but got ${array.constructor.name}`,
            });
        }
    }
}
export class ContiguityValidator {
    highWaterMarks;
    allowNoHighWaterMark = false;
    allowOverlap = false;
    allowGaps = false;
    constructor(props) {
        this.highWaterMarks = new Map();
        this.allowNoHighWaterMark = props.allowNoHighWaterMark;
        this.allowOverlap = props.allowOverlap;
        this.allowGaps = props.allowGaps;
    }
    validate(segment) {
        if (!segment.channel.key) {
            throw new UnexpectedError('Channel key is not set');
        }
        const hwm = this.getHighWaterMark(segment.channel.key);
        if (hwm) {
            this.enforceNoOverlap(hwm, segment);
            this.enforceNoGaps(hwm, segment);
        }
        this.updateHighWaterMark(segment);
    }
    enforceNoOverlap(hwm, seg) {
        if (!this.allowOverlap && seg.start.before(hwm)) {
            throw new ContiguityError(`Segment overlaps with previous segment. Previous segment ends at ${hwm.toString()}
        Segment starts at ${seg.start.toString()}`);
        }
    }
    enforceNoGaps(hwm, seg) {
        if (!this.allowGaps && !seg.start.equals(hwm)) {
            throw new ContiguityError(`Segment is not contiguous with previous segment. Previous segment ends at ${hwm.toString()}
        Segment starts at ${seg.start.toString()}`);
        }
    }
    getHighWaterMark(key) {
        const hwm = this.highWaterMarks.get(key);
        if (!hwm && !this.allowNoHighWaterMark) {
            throw new UnexpectedError('No high water mark found for channel key ' + key);
        }
        return hwm;
    }
    updateHighWaterMark(seg) {
        if (!seg.channel.key) {
            throw new UnexpectedError('Channel key is not set');
        }
        this.highWaterMarks.set(seg.channel.key, seg.end);
    }
}
//# sourceMappingURL=data:application/json;base64,eyJ2ZXJzaW9uIjozLCJmaWxlIjoidmFsaWRhdG9yLmpzIiwic291cmNlUm9vdCI6IiIsInNvdXJjZXMiOlsiLi4vLi4vLi4vLi4vc3JjL2xpYi9zZWdtZW50L3ZhbGlkYXRvci50cyJdLCJuYW1lcyI6W10sIm1hcHBpbmdzIjoiQUFBQSxPQUFPLEVBQUUsZUFBZSxFQUFFLGVBQWUsRUFBRSxlQUFlLEVBQUUsTUFBTSxXQUFXLENBQUM7QUFLOUUsTUFBTSxPQUFPLG1CQUFtQjtJQUM5QixRQUFRLENBQUMsS0FBaUIsRUFBRSxRQUFrQjtRQUM1QyxJQUFJLENBQUMsUUFBUSxDQUFDLFVBQVUsQ0FBQyxLQUFLLENBQUMsRUFBRTtZQUMvQixNQUFNLElBQUksZUFBZSxDQUFDO2dCQUN4QixLQUFLLEVBQUUsTUFBTTtnQkFDYixPQUFPLEVBQUUsZ0NBQWdDLFFBQVEsWUFBWSxLQUFLLENBQUMsV0FBVyxDQUFDLElBQUksRUFBRTthQUN0RixDQUFDLENBQUM7U0FDSjtJQUNILENBQUM7Q0FDRjtBQVFELE1BQU0sT0FBTyxtQkFBbUI7SUFDOUIsY0FBYyxDQUF5QjtJQUN2QyxvQkFBb0IsR0FBRyxLQUFLLENBQUM7SUFDN0IsWUFBWSxHQUFHLEtBQUssQ0FBQztJQUNyQixTQUFTLEdBQUcsS0FBSyxDQUFDO0lBRWxCLFlBQVksS0FBK0I7UUFDekMsSUFBSSxDQUFDLGNBQWMsR0FBRyxJQUFJLEdBQUcsRUFBRSxDQUFDO1FBQ2hDLElBQUksQ0FBQyxvQkFBb0IsR0FBRyxLQUFLLENBQUMsb0JBQW9CLENBQUM7UUFDdkQsSUFBSSxDQUFDLFlBQVksR0FBRyxLQUFLLENBQUMsWUFBWSxDQUFDO1FBQ3ZDLElBQUksQ0FBQyxTQUFTLEdBQUcsS0FBSyxDQUFDLFNBQVMsQ0FBQztJQUNuQyxDQUFDO0lBRUQsUUFBUSxDQUFDLE9BQXFCO1FBQzVCLElBQUksQ0FBQyxPQUFPLENBQUMsT0FBTyxDQUFDLEdBQUcsRUFBRTtZQUN4QixNQUFNLElBQUksZUFBZSxDQUFDLHdCQUF3QixDQUFDLENBQUM7U0FDckQ7UUFDRCxNQUFNLEdBQUcsR0FBRyxJQUFJLENBQUMsZ0JBQWdCLENBQUMsT0FBTyxDQUFDLE9BQU8sQ0FBQyxHQUFHLENBQUMsQ0FBQztRQUN2RCxJQUFJLEdBQUcsRUFBRTtZQUNQLElBQUksQ0FBQyxnQkFBZ0IsQ0FBQyxHQUFHLEVBQUUsT0FBTyxDQUFDLENBQUM7WUFDcEMsSUFBSSxDQUFDLGFBQWEsQ0FBQyxHQUFHLEVBQUUsT0FBTyxDQUFDLENBQUM7U0FDbEM7UUFDRCxJQUFJLENBQUMsbUJBQW1CLENBQUMsT0FBTyxDQUFDLENBQUM7SUFDcEMsQ0FBQztJQUVPLGdCQUFnQixDQUFDLEdBQWMsRUFBRSxHQUFpQjtRQUN4RCxJQUFJLENBQUMsSUFBSSxDQUFDLFlBQVksSUFBSSxHQUFHLENBQUMsS0FBSyxDQUFDLE1BQU0sQ0FBQyxHQUFHLENBQUMsRUFBRTtZQUMvQyxNQUFNLElBQUksZUFBZSxDQUN2QixvRUFBb0UsR0FBRyxDQUFDLFFBQVEsRUFBRTs0QkFDOUQsR0FBRyxDQUFDLEtBQUssQ0FBQyxRQUFRLEVBQUUsRUFBRSxDQUMzQyxDQUFDO1NBQ0g7SUFDSCxDQUFDO0lBRU8sYUFBYSxDQUFDLEdBQWMsRUFBRSxHQUFpQjtRQUNyRCxJQUFJLENBQUMsSUFBSSxDQUFDLFNBQVMsSUFBSSxDQUFDLEdBQUcsQ0FBQyxLQUFLLENBQUMsTUFBTSxDQUFDLEdBQUcsQ0FBQyxFQUFFO1lBQzdDLE1BQU0sSUFBSSxlQUFlLENBQ3ZCLDZFQUE2RSxHQUFHLENBQUMsUUFBUSxFQUFFOzRCQUN2RSxHQUFHLENBQUMsS0FBSyxDQUFDLFFBQVEsRUFBRSxFQUFFLENBQzNDLENBQUM7U0FDSDtJQUNILENBQUM7SUFFTyxnQkFBZ0IsQ0FBQyxHQUFXO1FBQ2xDLE1BQU0sR0FBRyxHQUFHLElBQUksQ0FBQyxjQUFjLENBQUMsR0FBRyxDQUFDLEdBQUcsQ0FBQyxDQUFDO1FBQ3pDLElBQUksQ0FBQyxHQUFHLElBQUksQ0FBQyxJQUFJLENBQUMsb0JBQW9CLEVBQUU7WUFDdEMsTUFBTSxJQUFJLGVBQWUsQ0FDdkIsMkNBQTJDLEdBQUcsR0FBRyxDQUNsRCxDQUFDO1NBQ0g7UUFDRCxPQUFPLEdBQUcsQ0FBQztJQUNiLENBQUM7SUFFTyxtQkFBbUIsQ0FBQyxHQUFpQjtRQUMzQyxJQUFJLENBQUMsR0FBRyxDQUFDLE9BQU8sQ0FBQyxHQUFHLEVBQUU7WUFDcEIsTUFBTSxJQUFJLGVBQWUsQ0FBQyx3QkFBd0IsQ0FBQyxDQUFDO1NBQ3JEO1FBQ0QsSUFBSSxDQUFDLGNBQWMsQ0FBQyxHQUFHLENBQUMsR0FBRyxDQUFDLE9BQU8sQ0FBQyxHQUFHLEVBQUUsR0FBRyxDQUFDLEdBQUcsQ0FBQyxDQUFDO0lBQ3BELENBQUM7Q0FDRiJ9