func pid{
    kp f32
    ki f32
    kd f32
    setpoint f32
} (input u8) f32 {
    error := setpoint - measurement
    p := kp * error
    last_measurement_time $= measurement_time
    dt := measurement_time - last_measurement_time
    integral $= 0
    integral = integral + error * f32(dt)
    i := ki * integral
    last_error $= error
    derivative := (error - last_error) / f32(dt)
    d := kd * derivative
    return p + i + d
}

interval{period=100ms} -> pid{
    kp=1, 
    ki=0.1, 
    kd=0.01, 
    setpoint=50
} -> setpoint_cmd