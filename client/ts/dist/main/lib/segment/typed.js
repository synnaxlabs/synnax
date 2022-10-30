"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
const errors_1 = require("../errors");
const telem_1 = require("../telem");
class TypedSegment {
    constructor(channel, payload) {
        this.channel = channel;
        this.payload = payload;
        this.view = new this.channel.dataType.arrayConstructor(this.payload.data.buffer);
    }
    get start() {
        return this.payload.start;
    }
    get span() {
        return this.channel.rate.byteSpan(telem_1.Size.Bytes(this.view.byteLength), this.channel.density);
    }
    get range() {
        return this.start.spanRange(this.span);
    }
    get end() {
        return this.range.end;
    }
    get size() {
        return telem_1.Size.Bytes(this.view.byteLength);
    }
    extend(other) {
        if (other.channel.key !== this.channel.key) {
            throw new errors_1.ValidationError(`
        Cannot extend segment because channel keys mismatch.
        Segment Channel Key: ${this.channel.key}
        Other Segment Channel Key: ${other.channel.key}
      `);
        }
        else if (!this.end.equals(other.start)) {
            throw new errors_1.ContiguityError(`
      Cannot extend segment because segments are not contiguous.
      Segment End: ${this.end}
      Other Segment Start: ${other.start}
      `);
        }
        const newData = new Uint8Array(this.view.byteLength + other.view.byteLength);
        newData.set(this.payload.data, 0);
        newData.set(other.payload.data, this.view.byteLength);
        this.payload.data = newData;
        this.view = new this.channel.dataType.arrayConstructor(this.payload.data.buffer);
    }
}
exports.default = TypedSegment;
//# sourceMappingURL=data:application/json;base64,eyJ2ZXJzaW9uIjozLCJmaWxlIjoidHlwZWQuanMiLCJzb3VyY2VSb290IjoiIiwic291cmNlcyI6WyIuLi8uLi8uLi8uLi9zcmMvbGliL3NlZ21lbnQvdHlwZWQudHMiXSwibmFtZXMiOltdLCJtYXBwaW5ncyI6Ijs7QUFDQSxzQ0FBNkQ7QUFDN0Qsb0NBT2tCO0FBSWxCLE1BQXFCLFlBQVk7SUFLL0IsWUFBWSxPQUF1QixFQUFFLE9BQXVCO1FBQzFELElBQUksQ0FBQyxPQUFPLEdBQUcsT0FBTyxDQUFDO1FBQ3ZCLElBQUksQ0FBQyxPQUFPLEdBQUcsT0FBTyxDQUFDO1FBQ3ZCLElBQUksQ0FBQyxJQUFJLEdBQUcsSUFBSSxJQUFJLENBQUMsT0FBTyxDQUFDLFFBQVEsQ0FBQyxnQkFBZ0IsQ0FDcEQsSUFBSSxDQUFDLE9BQU8sQ0FBQyxJQUFJLENBQUMsTUFBTSxDQUN6QixDQUFDO0lBQ0osQ0FBQztJQUVELElBQUksS0FBSztRQUNQLE9BQU8sSUFBSSxDQUFDLE9BQU8sQ0FBQyxLQUFLLENBQUM7SUFDNUIsQ0FBQztJQUVELElBQUksSUFBSTtRQUNOLE9BQU8sSUFBSSxDQUFDLE9BQU8sQ0FBQyxJQUFJLENBQUMsUUFBUSxDQUMvQixZQUFJLENBQUMsS0FBSyxDQUFDLElBQUksQ0FBQyxJQUFJLENBQUMsVUFBVSxDQUFDLEVBQ2hDLElBQUksQ0FBQyxPQUFPLENBQUMsT0FBa0IsQ0FDaEMsQ0FBQztJQUNKLENBQUM7SUFFRCxJQUFJLEtBQUs7UUFDUCxPQUFPLElBQUksQ0FBQyxLQUFLLENBQUMsU0FBUyxDQUFDLElBQUksQ0FBQyxJQUFJLENBQUMsQ0FBQztJQUN6QyxDQUFDO0lBRUQsSUFBSSxHQUFHO1FBQ0wsT0FBTyxJQUFJLENBQUMsS0FBSyxDQUFDLEdBQUcsQ0FBQztJQUN4QixDQUFDO0lBRUQsSUFBSSxJQUFJO1FBQ04sT0FBTyxZQUFJLENBQUMsS0FBSyxDQUFDLElBQUksQ0FBQyxJQUFJLENBQUMsVUFBVSxDQUFDLENBQUM7SUFDMUMsQ0FBQztJQUVELE1BQU0sQ0FBQyxLQUFtQjtRQUN4QixJQUFJLEtBQUssQ0FBQyxPQUFPLENBQUMsR0FBRyxLQUFLLElBQUksQ0FBQyxPQUFPLENBQUMsR0FBRyxFQUFFO1lBQzFDLE1BQU0sSUFBSSx3QkFBZSxDQUFDOzsrQkFFRCxJQUFJLENBQUMsT0FBTyxDQUFDLEdBQUc7cUNBQ1YsS0FBSyxDQUFDLE9BQU8sQ0FBQyxHQUFHO09BQy9DLENBQUMsQ0FBQztTQUNKO2FBQU0sSUFBSSxDQUFDLElBQUksQ0FBQyxHQUFHLENBQUMsTUFBTSxDQUFDLEtBQUssQ0FBQyxLQUFLLENBQUMsRUFBRTtZQUN4QyxNQUFNLElBQUksd0JBQWUsQ0FBQzs7cUJBRVgsSUFBSSxDQUFDLEdBQUc7NkJBQ0EsS0FBSyxDQUFDLEtBQUs7T0FDakMsQ0FBQyxDQUFDO1NBQ0o7UUFDRCxNQUFNLE9BQU8sR0FBRyxJQUFJLFVBQVUsQ0FDNUIsSUFBSSxDQUFDLElBQUksQ0FBQyxVQUFVLEdBQUcsS0FBSyxDQUFDLElBQUksQ0FBQyxVQUFVLENBQzdDLENBQUM7UUFDRixPQUFPLENBQUMsR0FBRyxDQUFDLElBQUksQ0FBQyxPQUFPLENBQUMsSUFBSSxFQUFFLENBQUMsQ0FBQyxDQUFDO1FBQ2xDLE9BQU8sQ0FBQyxHQUFHLENBQUMsS0FBSyxDQUFDLE9BQU8sQ0FBQyxJQUFJLEVBQUUsSUFBSSxDQUFDLElBQUksQ0FBQyxVQUFVLENBQUMsQ0FBQztRQUN0RCxJQUFJLENBQUMsT0FBTyxDQUFDLElBQUksR0FBRyxPQUFPLENBQUM7UUFDNUIsSUFBSSxDQUFDLElBQUksR0FBRyxJQUFJLElBQUksQ0FBQyxPQUFPLENBQUMsUUFBUSxDQUFDLGdCQUFnQixDQUNwRCxJQUFJLENBQUMsT0FBTyxDQUFDLElBQUksQ0FBQyxNQUFNLENBQ3pCLENBQUM7SUFDSixDQUFDO0NBQ0Y7QUE1REQsK0JBNERDIn0=