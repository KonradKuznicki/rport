package refs

type Origin interface {
	Source() identifiable
	Parent() identifiable
	NextFromIdentifiable(nextParent identifiable) origin
	Next(iType IdentifiableType, id string) origin
	String() string
}

type origin struct {
	source identifiable
	parent identifiable
}

func (o origin) Source() identifiable {
	//TODO implement me
	panic("implement me")
}

func (o origin) Parent() identifiable {
	//TODO implement me
	panic("implement me")
}

func (o origin) NextFromIdentifiable(nextParent identifiable) origin {
	//TODO implement me
	panic("implement me")
}

func (o origin) Next(iType IdentifiableType, id string) origin {
	return origin{source: o.source, parent: NewIdentifiable(iType, id)}
}

func (o origin) String() string {
	return "Origin(Source(" + o.source.String() + "), Parent(" + o.parent.String() + "))"
}

func NewOrigin(source identifiable, parent identifiable) origin {
	return origin{source: source, parent: parent}
}
