// Test mixed continuous and one-shot operators in flow chains
func double(val f32) f32 {
    return val * 2
}

func check(val f32) u8 {
    return val > 100
}

// Mixed operator chain: continuous -> double => check
sensor -> double{} => check{}
