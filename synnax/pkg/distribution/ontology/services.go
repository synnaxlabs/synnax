package ontology

type Service interface {
	Schema() *Schema
	RetrieveEntity(key string) (Entity, error)
}

type services map[Type]Service

func (s services) Register(svc Service) {
	t := svc.Schema().Type
	if _, ok := s[t]; ok {
		panic("[ontology] - service already registered")
	}
	s[t] = svc
}

func (s services) RetrieveEntity(key ID) (Entity, error) {
	if key.Type == BuiltIn {
		return Entity{}, nil
	}
	svc, ok := s[key.Type]
	if !ok {
		panic("[ontology] - service not found")
	}
	return svc.RetrieveEntity(key.Key)
}
