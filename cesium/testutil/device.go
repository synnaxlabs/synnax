package testutil

import (
	"context"
	"github.com/arya-analytics/cesium"
	"github.com/arya-analytics/cesium/testutil/seg"
	log "github.com/sirupsen/logrus"
	"io"
	"time"
)

type DeviceFactory struct {
	Ctx context.Context
	DB  cesium.DB
}

func (f DeviceFactory) New(dt cesium.Density, dr cesium.DataRate, fi cesium.TimeSpan) *Device {
	return &Device{
		Ctx:           f.Ctx,
		DB:            f.DB,
		DataType:      dt,
		DataRate:      dr,
		FlushInterval: fi,
	}
}

type Device struct {
	Ctx           context.Context
	DB            cesium.DB
	DataType      cesium.Density
	DataRate      cesium.DataRate
	FlushInterval cesium.TimeSpan
	res           <-chan cesium.CreateResponse
	cancelFlush   context.CancelFunc
}

func (d *Device) createChannel() (cesium.Channel, error) {
	return d.DB.CreateChannel().
		WithRate(d.DataRate).
		WithType(d.DataType).
		Exec(d.Ctx)
}

func (d *Device) Start() error {
	c, err := d.createChannel()
	if err != nil {
		return err
	}
	return d.writeSegments(c)
}

func (d *Device) writeSegments(c cesium.Channel) error {
	req, res, err := d.DB.NewCreate().WhereChannels(c.Key).Stream(d.Ctx)
	if err != nil {
		return err

	}
	d.res = res
	ctx, cancel := context.WithCancel(d.Ctx)
	d.cancelFlush = cancel
	sc := &seg.StreamCreate{
		Req:               req,
		Res:               res,
		SequentialFactory: seg.NewSequentialFactory(seg.DataTypeFactory(d.DataType), d.FlushInterval, c),
	}
	go func() {
		t := time.NewTicker(d.FlushInterval.Duration())
		defer t.Stop()
		defer close(req)
		for {
			select {
			case <-ctx.Done():
				return
			case resV := <-res:
				if resV.Error != nil {
					log.Error(err)
				}
			case <-t.C:
				req <- cesium.CreateRequest{Segments: sc.NextN(1)}
			}
		}
	}()
	return nil

}

func (d *Device) Stop() error {
	d.cancelFlush()
	resV := <-d.res
	if resV.Error == io.EOF {
		return nil
	}
	return resV.Error

}
