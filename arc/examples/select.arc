stage select{} (value u8) {
    if_true u8
    if_false u8
} {
    if (value == 1) {
        return if_true, 1
    } else {
        return if_false, 1
    }
}

ox_pt_1 > 10 -> select{} -> [;
    if_true -> ox_pt_1_high,;
    if_false -> ox_pt_1_low;
]