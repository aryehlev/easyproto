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

// Oneof example types

// Message is an interface representing different message types
type Message interface {
	MessageType() string
}

// TextMessage represents a text message
type TextMessage struct {
	Text   string `protobuf:"1"`
	Author string `protobuf:"2"`
}

func (t *TextMessage) MessageType() string { return "text" }

// ImageMessage represents an image message
type ImageMessage struct {
	URL    string `protobuf:"1"`
	Width  int32  `protobuf:"2"`
	Height int32  `protobuf:"3"`
}

func (i *ImageMessage) MessageType() string { return "image" }

// VideoMessage represents a video message
type VideoMessage struct {
	URL      string `protobuf:"1"`
	Duration int64  `protobuf:"2"`
}

func (v *VideoMessage) MessageType() string { return "video" }

// ChatMessage uses oneof for polymorphic message content
type ChatMessage struct {
	ID      int64   `protobuf:"1"`
	Content Message `protobuf:"oneof,TextMessage:2,ImageMessage:3,VideoMessage:4"`
}

// ChatHistory contains multiple chat messages with oneof
type ChatHistory struct {
	Title    string         `protobuf:"1"`
	Messages []*ChatMessage `protobuf:"2"`
}
