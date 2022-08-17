package pool

type Demand int64

func (d *Demand) Increase(amount Demand) { *d += amount }

func (d *Demand) Decrease(amount Demand) { *d -= amount }
