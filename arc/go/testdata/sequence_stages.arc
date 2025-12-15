// Test sequence with stage targets and transitions
func threshold(val f32) u8 {
    return val > 100
}

 func prepare() u8 {
    return 1
}

func recover() u8 {
    return 1
}

sequence main {
    stage initialization {
        // Continuous flow to next stage
        sensor -> prepare{} => next
    }

    stage pressurization {
        // One-shot transitions with check
        sensor -> threshold{} => next,
        pressure -> threshold{} => abort
    }

    stage abort {
        // Can transition back
        recover{} => initialization
    }
}

// Top-level flow using one-shot operator
start_cmd => main
