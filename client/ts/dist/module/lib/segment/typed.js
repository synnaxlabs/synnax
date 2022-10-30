import { ContiguityError, ValidationError } from '../errors';
import { Size, } from '../telem';
export default class TypedSegment {
    payload;
    channel;
    view;
    constructor(channel, payload) {
        this.channel = channel;
        this.payload = payload;
        this.view = new this.channel.dataType.arrayConstructor(this.payload.data.buffer);
    }
    get start() {
        return this.payload.start;
    }
    get span() {
        return this.channel.rate.byteSpan(Size.Bytes(this.view.byteLength), this.channel.density);
    }
    get range() {
        return this.start.spanRange(this.span);
    }
    get end() {
        return this.range.end;
    }
    get size() {
        return Size.Bytes(this.view.byteLength);
    }
    extend(other) {
        if (other.channel.key !== this.channel.key) {
            throw new ValidationError(`
        Cannot extend segment because channel keys mismatch.
        Segment Channel Key: ${this.channel.key}
        Other Segment Channel Key: ${other.channel.key}
      `);
        }
        else if (!this.end.equals(other.start)) {
            throw new ContiguityError(`
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
//# sourceMappingURL=data:application/json;base64,eyJ2ZXJzaW9uIjozLCJmaWxlIjoidHlwZWQuanMiLCJzb3VyY2VSb290IjoiIiwic291cmNlcyI6WyIuLi8uLi8uLi8uLi9zcmMvbGliL3NlZ21lbnQvdHlwZWQudHMiXSwibmFtZXMiOltdLCJtYXBwaW5ncyI6IkFBQ0EsT0FBTyxFQUFFLGVBQWUsRUFBRSxlQUFlLEVBQUUsTUFBTSxXQUFXLENBQUM7QUFDN0QsT0FBTyxFQUVMLElBQUksR0FLTCxNQUFNLFVBQVUsQ0FBQztBQUlsQixNQUFNLENBQUMsT0FBTyxPQUFPLFlBQVk7SUFDL0IsT0FBTyxDQUFpQjtJQUN4QixPQUFPLENBQWlCO0lBQ3hCLElBQUksQ0FBYTtJQUVqQixZQUFZLE9BQXVCLEVBQUUsT0FBdUI7UUFDMUQsSUFBSSxDQUFDLE9BQU8sR0FBRyxPQUFPLENBQUM7UUFDdkIsSUFBSSxDQUFDLE9BQU8sR0FBRyxPQUFPLENBQUM7UUFDdkIsSUFBSSxDQUFDLElBQUksR0FBRyxJQUFJLElBQUksQ0FBQyxPQUFPLENBQUMsUUFBUSxDQUFDLGdCQUFnQixDQUNwRCxJQUFJLENBQUMsT0FBTyxDQUFDLElBQUksQ0FBQyxNQUFNLENBQ3pCLENBQUM7SUFDSixDQUFDO0lBRUQsSUFBSSxLQUFLO1FBQ1AsT0FBTyxJQUFJLENBQUMsT0FBTyxDQUFDLEtBQUssQ0FBQztJQUM1QixDQUFDO0lBRUQsSUFBSSxJQUFJO1FBQ04sT0FBTyxJQUFJLENBQUMsT0FBTyxDQUFDLElBQUksQ0FBQyxRQUFRLENBQy9CLElBQUksQ0FBQyxLQUFLLENBQUMsSUFBSSxDQUFDLElBQUksQ0FBQyxVQUFVLENBQUMsRUFDaEMsSUFBSSxDQUFDLE9BQU8sQ0FBQyxPQUFrQixDQUNoQyxDQUFDO0lBQ0osQ0FBQztJQUVELElBQUksS0FBSztRQUNQLE9BQU8sSUFBSSxDQUFDLEtBQUssQ0FBQyxTQUFTLENBQUMsSUFBSSxDQUFDLElBQUksQ0FBQyxDQUFDO0lBQ3pDLENBQUM7SUFFRCxJQUFJLEdBQUc7UUFDTCxPQUFPLElBQUksQ0FBQyxLQUFLLENBQUMsR0FBRyxDQUFDO0lBQ3hCLENBQUM7SUFFRCxJQUFJLElBQUk7UUFDTixPQUFPLElBQUksQ0FBQyxLQUFLLENBQUMsSUFBSSxDQUFDLElBQUksQ0FBQyxVQUFVLENBQUMsQ0FBQztJQUMxQyxDQUFDO0lBRUQsTUFBTSxDQUFDLEtBQW1CO1FBQ3hCLElBQUksS0FBSyxDQUFDLE9BQU8sQ0FBQyxHQUFHLEtBQUssSUFBSSxDQUFDLE9BQU8sQ0FBQyxHQUFHLEVBQUU7WUFDMUMsTUFBTSxJQUFJLGVBQWUsQ0FBQzs7K0JBRUQsSUFBSSxDQUFDLE9BQU8sQ0FBQyxHQUFHO3FDQUNWLEtBQUssQ0FBQyxPQUFPLENBQUMsR0FBRztPQUMvQyxDQUFDLENBQUM7U0FDSjthQUFNLElBQUksQ0FBQyxJQUFJLENBQUMsR0FBRyxDQUFDLE1BQU0sQ0FBQyxLQUFLLENBQUMsS0FBSyxDQUFDLEVBQUU7WUFDeEMsTUFBTSxJQUFJLGVBQWUsQ0FBQzs7cUJBRVgsSUFBSSxDQUFDLEdBQUc7NkJBQ0EsS0FBSyxDQUFDLEtBQUs7T0FDakMsQ0FBQyxDQUFDO1NBQ0o7UUFDRCxNQUFNLE9BQU8sR0FBRyxJQUFJLFVBQVUsQ0FDNUIsSUFBSSxDQUFDLElBQUksQ0FBQyxVQUFVLEdBQUcsS0FBSyxDQUFDLElBQUksQ0FBQyxVQUFVLENBQzdDLENBQUM7UUFDRixPQUFPLENBQUMsR0FBRyxDQUFDLElBQUksQ0FBQyxPQUFPLENBQUMsSUFBSSxFQUFFLENBQUMsQ0FBQyxDQUFDO1FBQ2xDLE9BQU8sQ0FBQyxHQUFHLENBQUMsS0FBSyxDQUFDLE9BQU8sQ0FBQyxJQUFJLEVBQUUsSUFBSSxDQUFDLElBQUksQ0FBQyxVQUFVLENBQUMsQ0FBQztRQUN0RCxJQUFJLENBQUMsT0FBTyxDQUFDLElBQUksR0FBRyxPQUFPLENBQUM7UUFDNUIsSUFBSSxDQUFDLElBQUksR0FBRyxJQUFJLElBQUksQ0FBQyxPQUFPLENBQUMsUUFBUSxDQUFDLGdCQUFnQixDQUNwRCxJQUFJLENBQUMsT0FBTyxDQUFDLElBQUksQ0FBQyxNQUFNLENBQ3pCLENBQUM7SUFDSixDQUFDO0NBQ0YifQ==