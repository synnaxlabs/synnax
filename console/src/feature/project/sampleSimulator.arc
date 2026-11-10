import time

// Simulates a tank that fills from a 500 psi nitrogen supply through the press valve
// and empties through the vent valve.
func step() {
    pressure f64 $= 0
    if sample_press_vlv_cmd == 1 {
        pressure += (500 - pressure) * 0.02
    }
    if sample_vent_vlv_cmd == 1 {
        pressure -= pressure * 0.05
    }
    pressure -= pressure * 0.0002
    sample_tank_pt = pressure
    sample_press_vlv_state = sample_press_vlv_cmd
    sample_vent_vlv_state = sample_vent_vlv_cmd
}

time.interval(period=100ms) -> step()
