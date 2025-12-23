package example

type a A

type A struct {
	B `protobuf:"1"`
}

type B struct {
	b string `protobuf:"1"`
}

type c any

type D interface {
	DoSomething() string
}

type d1 struct {
	ddd d `protobuf:"1"`
}

type d2 struct {
	d `protobuf:"1"`
}
type d struct {
	dd string `protobuf:"1"`
}

func (d *d) DoSomething() string {
	return d.dd
}

type all struct {
	d `protobuf:"1"`
	a A `protobuf:"2"`
	c c `protobuf:"3"`
}

type all1 all

type all2 struct {
	all all1 `protobuf:"1"`
}

type all3 struct {
	all1 `protobuf:"1"`
	aa   map[string]all2 `protobuf:"2"`
	bb   []all1          `protobuf:"3"`
}
