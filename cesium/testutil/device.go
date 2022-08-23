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

func (f DeviceFactory) New(dt cesium.Density, dr cesium.Rate, fi cesium.TimeSpan) *Device {
	return &Device{
		Ctx:           f.Ctx,
		DB:            f.DB,
		Density:       dt,
		Rate:          dr,
		FlushInterval: fi,
	}
}

type Device struct {
	Ctx           context.Context
	DB            cesium.DB
	Density       cesium.Density
	Rate          cesium.Rate
	FlushInterval cesium.TimeSpan
	res           <-chan cesium.CreateResponse
	cancelFlush   context.CancelFunc
}

func (d *Device) createChannel() (cesium.Channel, error) {
	ch := cesium.Channel{
		Rate:    d.Rate,
		Density: d.Density,
	}
	key, err := d.DB.CreateChannel(ch)
	ch.Key = key
	return ch, err
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
		SequentialFactory: seg.NewSequentialFactory(seg.DensityFactory(d.Density), d.FlushInterval, c),
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
