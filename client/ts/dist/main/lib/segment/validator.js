"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.ContiguityValidator = exports.ScalarTypeValidator = void 0;
const errors_1 = require("../errors");
class ScalarTypeValidator {
    validate(array, dataType) {
        if (!dataType.checkArray(array)) {
            throw new errors_1.ValidationError({
                field: 'data',
                message: `Data type mismatch. Expected ${dataType} but got ${array.constructor.name}`,
            });
        }
    }
}
exports.ScalarTypeValidator = ScalarTypeValidator;
class ContiguityValidator {
    constructor(props) {
        this.allowNoHighWaterMark = false;
        this.allowOverlap = false;
        this.allowGaps = false;
        this.highWaterMarks = new Map();
        this.allowNoHighWaterMark = props.allowNoHighWaterMark;
        this.allowOverlap = props.allowOverlap;
        this.allowGaps = props.allowGaps;
    }
    validate(segment) {
        if (!segment.channel.key) {
            throw new errors_1.UnexpectedError('Channel key is not set');
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
            throw new errors_1.ContiguityError(`Segment overlaps with previous segment. Previous segment ends at ${hwm.toString()}
        Segment starts at ${seg.start.toString()}`);
        }
    }
    enforceNoGaps(hwm, seg) {
        if (!this.allowGaps && !seg.start.equals(hwm)) {
            throw new errors_1.ContiguityError(`Segment is not contiguous with previous segment. Previous segment ends at ${hwm.toString()}
        Segment starts at ${seg.start.toString()}`);
        }
    }
    getHighWaterMark(key) {
        const hwm = this.highWaterMarks.get(key);
        if (!hwm && !this.allowNoHighWaterMark) {
            throw new errors_1.UnexpectedError('No high water mark found for channel key ' + key);
        }
        return hwm;
    }
    updateHighWaterMark(seg) {
        if (!seg.channel.key) {
            throw new errors_1.UnexpectedError('Channel key is not set');
        }
        this.highWaterMarks.set(seg.channel.key, seg.end);
    }
}
exports.ContiguityValidator = ContiguityValidator;
//# sourceMappingURL=data:application/json;base64,eyJ2ZXJzaW9uIjozLCJmaWxlIjoidmFsaWRhdG9yLmpzIiwic291cmNlUm9vdCI6IiIsInNvdXJjZXMiOlsiLi4vLi4vLi4vLi4vc3JjL2xpYi9zZWdtZW50L3ZhbGlkYXRvci50cyJdLCJuYW1lcyI6W10sIm1hcHBpbmdzIjoiOzs7QUFBQSxzQ0FBOEU7QUFLOUUsTUFBYSxtQkFBbUI7SUFDOUIsUUFBUSxDQUFDLEtBQWlCLEVBQUUsUUFBa0I7UUFDNUMsSUFBSSxDQUFDLFFBQVEsQ0FBQyxVQUFVLENBQUMsS0FBSyxDQUFDLEVBQUU7WUFDL0IsTUFBTSxJQUFJLHdCQUFlLENBQUM7Z0JBQ3hCLEtBQUssRUFBRSxNQUFNO2dCQUNiLE9BQU8sRUFBRSxnQ0FBZ0MsUUFBUSxZQUFZLEtBQUssQ0FBQyxXQUFXLENBQUMsSUFBSSxFQUFFO2FBQ3RGLENBQUMsQ0FBQztTQUNKO0lBQ0gsQ0FBQztDQUNGO0FBVEQsa0RBU0M7QUFRRCxNQUFhLG1CQUFtQjtJQU05QixZQUFZLEtBQStCO1FBSjNDLHlCQUFvQixHQUFHLEtBQUssQ0FBQztRQUM3QixpQkFBWSxHQUFHLEtBQUssQ0FBQztRQUNyQixjQUFTLEdBQUcsS0FBSyxDQUFDO1FBR2hCLElBQUksQ0FBQyxjQUFjLEdBQUcsSUFBSSxHQUFHLEVBQUUsQ0FBQztRQUNoQyxJQUFJLENBQUMsb0JBQW9CLEdBQUcsS0FBSyxDQUFDLG9CQUFvQixDQUFDO1FBQ3ZELElBQUksQ0FBQyxZQUFZLEdBQUcsS0FBSyxDQUFDLFlBQVksQ0FBQztRQUN2QyxJQUFJLENBQUMsU0FBUyxHQUFHLEtBQUssQ0FBQyxTQUFTLENBQUM7SUFDbkMsQ0FBQztJQUVELFFBQVEsQ0FBQyxPQUFxQjtRQUM1QixJQUFJLENBQUMsT0FBTyxDQUFDLE9BQU8sQ0FBQyxHQUFHLEVBQUU7WUFDeEIsTUFBTSxJQUFJLHdCQUFlLENBQUMsd0JBQXdCLENBQUMsQ0FBQztTQUNyRDtRQUNELE1BQU0sR0FBRyxHQUFHLElBQUksQ0FBQyxnQkFBZ0IsQ0FBQyxPQUFPLENBQUMsT0FBTyxDQUFDLEdBQUcsQ0FBQyxDQUFDO1FBQ3ZELElBQUksR0FBRyxFQUFFO1lBQ1AsSUFBSSxDQUFDLGdCQUFnQixDQUFDLEdBQUcsRUFBRSxPQUFPLENBQUMsQ0FBQztZQUNwQyxJQUFJLENBQUMsYUFBYSxDQUFDLEdBQUcsRUFBRSxPQUFPLENBQUMsQ0FBQztTQUNsQztRQUNELElBQUksQ0FBQyxtQkFBbUIsQ0FBQyxPQUFPLENBQUMsQ0FBQztJQUNwQyxDQUFDO0lBRU8sZ0JBQWdCLENBQUMsR0FBYyxFQUFFLEdBQWlCO1FBQ3hELElBQUksQ0FBQyxJQUFJLENBQUMsWUFBWSxJQUFJLEdBQUcsQ0FBQyxLQUFLLENBQUMsTUFBTSxDQUFDLEdBQUcsQ0FBQyxFQUFFO1lBQy9DLE1BQU0sSUFBSSx3QkFBZSxDQUN2QixvRUFBb0UsR0FBRyxDQUFDLFFBQVEsRUFBRTs0QkFDOUQsR0FBRyxDQUFDLEtBQUssQ0FBQyxRQUFRLEVBQUUsRUFBRSxDQUMzQyxDQUFDO1NBQ0g7SUFDSCxDQUFDO0lBRU8sYUFBYSxDQUFDLEdBQWMsRUFBRSxHQUFpQjtRQUNyRCxJQUFJLENBQUMsSUFBSSxDQUFDLFNBQVMsSUFBSSxDQUFDLEdBQUcsQ0FBQyxLQUFLLENBQUMsTUFBTSxDQUFDLEdBQUcsQ0FBQyxFQUFFO1lBQzdDLE1BQU0sSUFBSSx3QkFBZSxDQUN2Qiw2RUFBNkUsR0FBRyxDQUFDLFFBQVEsRUFBRTs0QkFDdkUsR0FBRyxDQUFDLEtBQUssQ0FBQyxRQUFRLEVBQUUsRUFBRSxDQUMzQyxDQUFDO1NBQ0g7SUFDSCxDQUFDO0lBRU8sZ0JBQWdCLENBQUMsR0FBVztRQUNsQyxNQUFNLEdBQUcsR0FBRyxJQUFJLENBQUMsY0FBYyxDQUFDLEdBQUcsQ0FBQyxHQUFHLENBQUMsQ0FBQztRQUN6QyxJQUFJLENBQUMsR0FBRyxJQUFJLENBQUMsSUFBSSxDQUFDLG9CQUFvQixFQUFFO1lBQ3RDLE1BQU0sSUFBSSx3QkFBZSxDQUN2QiwyQ0FBMkMsR0FBRyxHQUFHLENBQ2xELENBQUM7U0FDSDtRQUNELE9BQU8sR0FBRyxDQUFDO0lBQ2IsQ0FBQztJQUVPLG1CQUFtQixDQUFDLEdBQWlCO1FBQzNDLElBQUksQ0FBQyxHQUFHLENBQUMsT0FBTyxDQUFDLEdBQUcsRUFBRTtZQUNwQixNQUFNLElBQUksd0JBQWUsQ0FBQyx3QkFBd0IsQ0FBQyxDQUFDO1NBQ3JEO1FBQ0QsSUFBSSxDQUFDLGNBQWMsQ0FBQyxHQUFHLENBQUMsR0FBRyxDQUFDLE9BQU8sQ0FBQyxHQUFHLEVBQUUsR0FBRyxDQUFDLEdBQUcsQ0FBQyxDQUFDO0lBQ3BELENBQUM7Q0FDRjtBQTNERCxrREEyREMifQ==