"use strict";
var __importDefault = (this && this.__importDefault) || function (mod) {
    return (mod && mod.__esModule) ? mod : { "default": mod };
};
Object.defineProperty(exports, "__esModule", { value: true });
const errors_1 = require("../errors");
const typed_1 = __importDefault(require("./typed"));
class Splitter {
    constructor(threshold) {
        this.threshold = threshold;
    }
    split(segment) {
        if (segment.size.smallerThan(this.threshold))
            return [segment];
        if (!segment.channel.density)
            throw new errors_1.ValidationError('Cannot split segment because channel density is undefined');
        const splitPoint = this.threshold.valueOf() -
            (this.threshold.valueOf() % segment.channel.density.valueOf());
        const truncated = new typed_1.default(segment.channel, Object.assign(Object.assign({}, segment.payload), { data: segment.payload.data.slice(0, splitPoint) }));
        const next = new typed_1.default(segment.channel, Object.assign(Object.assign({}, segment.payload), { data: segment.payload.data.slice(splitPoint) }));
        return [truncated, ...this.split(next)];
    }
}
exports.default = Splitter;
//# sourceMappingURL=data:application/json;base64,eyJ2ZXJzaW9uIjozLCJmaWxlIjoic3BsaXR0ZXIuanMiLCJzb3VyY2VSb290IjoiIiwic291cmNlcyI6WyIuLi8uLi8uLi8uLi9zcmMvbGliL3NlZ21lbnQvc3BsaXR0ZXIudHMiXSwibmFtZXMiOltdLCJtYXBwaW5ncyI6Ijs7Ozs7QUFBQSxzQ0FBNEM7QUFHNUMsb0RBQW1DO0FBRW5DLE1BQXFCLFFBQVE7SUFHM0IsWUFBWSxTQUFlO1FBQ3pCLElBQUksQ0FBQyxTQUFTLEdBQUcsU0FBUyxDQUFDO0lBQzdCLENBQUM7SUFFRCxLQUFLLENBQUMsT0FBcUI7UUFDekIsSUFBSSxPQUFPLENBQUMsSUFBSSxDQUFDLFdBQVcsQ0FBQyxJQUFJLENBQUMsU0FBUyxDQUFDO1lBQUUsT0FBTyxDQUFDLE9BQU8sQ0FBQyxDQUFDO1FBQy9ELElBQUksQ0FBQyxPQUFPLENBQUMsT0FBTyxDQUFDLE9BQU87WUFDMUIsTUFBTSxJQUFJLHdCQUFlLENBQ3ZCLDJEQUEyRCxDQUM1RCxDQUFDO1FBQ0osTUFBTSxVQUFVLEdBQ2QsSUFBSSxDQUFDLFNBQVMsQ0FBQyxPQUFPLEVBQUU7WUFDeEIsQ0FBQyxJQUFJLENBQUMsU0FBUyxDQUFDLE9BQU8sRUFBRSxHQUFHLE9BQU8sQ0FBQyxPQUFPLENBQUMsT0FBTyxDQUFDLE9BQU8sRUFBRSxDQUFDLENBQUM7UUFDakUsTUFBTSxTQUFTLEdBQUcsSUFBSSxlQUFZLENBQUMsT0FBTyxDQUFDLE9BQU8sa0NBQzdDLE9BQU8sQ0FBQyxPQUFPLEtBQ2xCLElBQUksRUFBRSxPQUFPLENBQUMsT0FBTyxDQUFDLElBQUksQ0FBQyxLQUFLLENBQUMsQ0FBQyxFQUFFLFVBQVUsQ0FBQyxJQUMvQyxDQUFDO1FBQ0gsTUFBTSxJQUFJLEdBQUcsSUFBSSxlQUFZLENBQUMsT0FBTyxDQUFDLE9BQU8sa0NBQ3hDLE9BQU8sQ0FBQyxPQUFPLEtBQ2xCLElBQUksRUFBRSxPQUFPLENBQUMsT0FBTyxDQUFDLElBQUksQ0FBQyxLQUFLLENBQUMsVUFBVSxDQUFDLElBQzVDLENBQUM7UUFDSCxPQUFPLENBQUMsU0FBUyxFQUFFLEdBQUcsSUFBSSxDQUFDLEtBQUssQ0FBQyxJQUFJLENBQUMsQ0FBQyxDQUFDO0lBQzFDLENBQUM7Q0FDRjtBQTFCRCwyQkEwQkMifQ==