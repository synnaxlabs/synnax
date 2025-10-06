stage max{} (value f32) f32 {
    max_val $= value
    if (value > max_val) {
        max_val = value
    }
    return max_val
}

stage min{
    reset chan u8
} (value f32) f32 {
    min_val $= value
    if (value < min_val) {
        min_val = value
    }
    if (reset) {
        min_val = value
    }
    return min_val
}

stage avg{} (value f32) f32 {
    sum_v f32 $= 0.0
    count $= 0
    sum_v = sum_v + value
    count = count + 1
    return sum_v / f32(count)
}

stage sum{} (value f32) f32 {
    total f32 $= 0.0
    total = total + value
    return total
}

stage counter{} () u32 {
    count u32 $= 0
    count = count + 1
    return count
}

stage ge{} (a f32, b f32) u8 {
    return a >= b
}

stage le{} (a f32, b f32) u8 {
    return a <= b
} 

stage select{} (cond f32) {
    if_true f32
    if_false f32
} {
    if (cond > 10) {
        if_true = cond
    } else {
        if_false = cond
    }
}

select{} -> {
    if_true -> ge{},
    if_false -> le{}
}