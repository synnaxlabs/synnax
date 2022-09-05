package grpc

//type StreamService struct {
//	segmentv1.UnimplementedIteratorServiceServer
//	segmentv1.UnimplementedWriterServiceServer
//	api.SegmentService
//}
//
//func (s *StreamService) Write(server segmentv1.IteratorService_IterateServer) error {
//	return s.SegmentService.Write(server.Context(), &iteratorServer{server})
//}
//
//func (s *StreamService) Write(server segmentv1.WriterService_WriteServer) error {
//	return s.SegmentService.Write(server.Context(), &writerServer{server})
//}
//
//type iteratorServer struct {
//	base segmentv1.IteratorService_IterateServer
//}
//
//func (i *iteratorServer) Receive() (api.IteratorRequest, error) {
//	req, err := i.base.Recv()
//	if err != nil {
//		return api.IteratorRequest{}, err
//	}
//	return i.translateRequest(req)
//}
//
//func (i *iteratorServer) Send(res api.IteratorResponse) error {
//	_res, err := i.translateResponse(res)
//	if err != nil {
//		return err
//	}
//	return i.base.Send(_res)
//}
//
//func (i *iteratorServer) translateRequest(req *segmentv1.IteratorRequest) (api.IteratorRequest, error) {
//	keys, err := channel.ParseKeys(req.ChannelKeys)
//	return api.IteratorRequest{
//		Command: api.IteratorCommand(req.Command),
//		SpanTo:    telem.TimeSpan(req.SpanTo),
//		TimeRange: telem.TimeRange{
//			Start: telem.TimeStamp(req.TimeRange.Start),
//			End:   telem.TimeStamp(req.TimeRange.End),
//		},
//		Stamp: telem.TimeStamp(req.Stamp),
//		ChannelKeys:  keys,
//	}, err
//}
//
//func (i *iteratorServer) translateResponse(res api.IteratorResponse) (*segmentv1.IteratorResponse, error) {
//	seg, err := translateSegmentsBackward(res.Segments)
//	return &segmentv1.IteratorResponse{Err: TranslateResponseForward(res.Err), Segments: seg}, err
//}
//
//type writerServer struct {
//	base segmentv1.WriterService_WriteServer
//}
//
//func (w *writerServer) Receive() (api.WriterRequest, error) {
//	req, err := w.base.Recv()
//	if err != nil {
//		return api.WriterRequest{}, err
//	}
//	return w.translateRequest(req)
//}
//
//func (w *writerServer) Send(resp api.WriterResponse) error {
//	return w.base.Send(w.translateResponse(resp))
//}
//
//func (w *writerServer) translateRequest(req *segmentv1.WriterRequest) (api.WriterRequest, error) {
//	seg, err := translateSegmentsForward(req.Segments)
//	if err != nil {
//		return api.WriterRequest{}, err
//	}
//	return api.WriterRequest{Segments: seg}, err
//}
//
//func (w *writerServer) translateResponse(res api.WriterResponse) *segmentv1.WriterResponse {
//	return &segmentv1.WriterResponse{Err: TranslateResponseForward(res.Err)}
//}
//
//func translateSegmentsBackward(segments []api.segment) ([]*segmentv1.segment, error) {
//	tSegments := make([]*segmentv1.segment, len(segments))
//	for i, s := range segments {
//		tSegments[i] = &segmentv1.segment{ChannelKey: s.ChannelKey.Report(), Start: int64(s.Start), Data: s.Data}
//	}
//	return tSegments, nil
//}
//
//func translateSegmentsForward(segments []*segmentv1.segment) ([]api.segment, error) {
//	tSegments := make([]api.segment, len(segments))
//	for i, s := range segments {
//		channelKey, err := channel.ParseKey(s.ChannelKey)
//		if err != nil {
//			return nil, err
//		}
//		tSegments[i] = api.segment{ChannelKey: channelKey, Start: telem.TimeStamp(s.Start), Data: s.Data}
//	}
//	return tSegments, nil
//}
