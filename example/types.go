// Package example demonstrates protogen usage with comprehensive edge cases.
package example

//go:generate go run ../cmd/protogen -type=Timeseries,Sample,AllTypes,OptionalFields,NestedMessage,WithPointers,WithEnums,RepeatedScalars,RepeatedStringsBytes,WithMaps,InferredTypes,BigStruct,SmallStruct

// Status is an enum type for demonstration.
type Status int32

const (
	StatusUnknown Status = 0
	StatusActive  Status = 1
	StatusPaused  Status = 2
	StatusStopped Status = 3
)

// Timeseries is a named time series (basic example from docs).
//
// protobuf definition:
//
//	message Timeseries {
//	  string name = 1;
//	  repeated Sample samples = 2;
//	}
type Timeseries struct {
	Name    string   `protobuf:"1,string"`
	Samples []Sample `protobuf:"2,message"`
}

// Sample represents a sample for the named time series.
//
// protobuf definition:
//
//	message Sample {
//	  double value = 1;
//	  int64 timestamp = 2;
//	}
type Sample struct {
	Value     float64 `protobuf:"1,double"`
	Timestamp int64   `protobuf:"2,int64"`
}

// AllTypes demonstrates all supported scalar types.
//
// protobuf definition:
//
//	message AllTypes {
//	  string str_field = 1;
//	  bytes bytes_field = 2;
//	  int32 int32_field = 3;
//	  int64 int64_field = 4;
//	  uint32 uint32_field = 5;
//	  uint64 uint64_field = 6;
//	  sint32 sint32_field = 7;
//	  sint64 sint64_field = 8;
//	  bool bool_field = 9;
//	  double double_field = 10;
//	  float float_field = 11;
//	  fixed32 fixed32_field = 12;
//	  fixed64 fixed64_field = 13;
//	  sfixed32 sfixed32_field = 14;
//	  sfixed64 sfixed64_field = 15;
//	}
type AllTypes struct {
	StrField      string  `protobuf:"1,string"`
	BytesField    []byte  `protobuf:"2,bytes"`
	Int32Field    int32   `protobuf:"3,int32"`
	Int64Field    int64   `protobuf:"4,int64"`
	Uint32Field   uint32  `protobuf:"5,uint32"`
	Uint64Field   uint64  `protobuf:"6,uint64"`
	Sint32Field   int32   `protobuf:"7,sint32"`
	Sint64Field   int64   `protobuf:"8,sint64"`
	BoolField     bool    `protobuf:"9,bool"`
	DoubleField   float64 `protobuf:"10,double"`
	FloatField    float32 `protobuf:"11,float"`
	Fixed32Field  uint32  `protobuf:"12,fixed32"`
	Fixed64Field  uint64  `protobuf:"13,fixed64"`
	Sfixed32Field int32   `protobuf:"14,sfixed32"`
	Sfixed64Field int64   `protobuf:"15,sfixed64"`
}

// OptionalFields demonstrates optional (pointer) fields.
//
// In proto3, optional fields are represented as pointers.
type OptionalFields struct {
	Name     *string  `protobuf:"1,string"`
	Age      *int32   `protobuf:"2,int32"`
	Score    *float64 `protobuf:"3,double"`
	IsActive *bool    `protobuf:"4,bool"`
}

// NestedMessage demonstrates nested message handling.
type NestedMessage struct {
	ID       int64    `protobuf:"1,int64"`
	Outer    Sample   `protobuf:"2,message"`          // Embedded message (value)
	Optional *Sample  `protobuf:"3,message,optional"` // Optional message (pointer)
	Items    []Sample `protobuf:"4,message"`          // Repeated message (slice of values)
}

// WithPointers demonstrates slice of pointers to messages.
type WithPointers struct {
	ID      int64     `protobuf:"1,int64"`
	Samples []*Sample `protobuf:"2,message"` // Slice of pointers to messages
}

// WithEnums demonstrates enum field handling.
type WithEnums struct {
	ID        int64    `protobuf:"1,int64"`
	Status    Status   `protobuf:"2,enum"`          // Regular enum
	OptStatus *Status  `protobuf:"3,enum,optional"` // Optional enum (pointer)
	Statuses  []Status `protobuf:"4,enum,repeated"` // Repeated enum
}

// RepeatedScalars demonstrates repeated scalar types.
//
// protobuf definition:
//
//	message RepeatedScalars {
//	  repeated int32 int32s = 1;
//	  repeated int64 int64s = 2;
//	  repeated uint32 uint32s = 3;
//	  repeated uint64 uint64s = 4;
//	  repeated sint32 sint32s = 5;
//	  repeated sint64 sint64s = 6;
//	  repeated bool bools = 7;
//	  repeated double doubles = 8;
//	  repeated float floats = 9;
//	  repeated fixed32 fixed32s = 10;
//	  repeated fixed64 fixed64s = 11;
//	  repeated sfixed32 sfixed32s = 12;
//	  repeated sfixed64 sfixed64s = 13;
//	}
type RepeatedScalars struct {
	Int32s    []int32   `protobuf:"1,int32"`
	Int64s    []int64   `protobuf:"2,int64"`
	Uint32s   []uint32  `protobuf:"3,uint32"`
	Uint64s   []uint64  `protobuf:"4,uint64"`
	Sint32s   []int32   `protobuf:"5,sint32"`
	Sint64s   []int64   `protobuf:"6,sint64"`
	Bools     []bool    `protobuf:"7,bool"`
	Doubles   []float64 `protobuf:"8,double"`
	Floats    []float32 `protobuf:"9,float"`
	Fixed32s  []uint32  `protobuf:"10,fixed32"`
	Fixed64s  []uint64  `protobuf:"11,fixed64"`
	Sfixed32s []int32   `protobuf:"12,sfixed32"`
	Sfixed64s []int64   `protobuf:"13,sfixed64"`
}

// RepeatedStringsBytes demonstrates repeated string and bytes fields.
// These are length-delimited types that cannot be packed.
type RepeatedStringsBytes struct {
	Strings []string `protobuf:"1,string"`
	Blobs   [][]byte `protobuf:"2,bytes"`
}

// WithMaps demonstrates map field handling.
//
// In protobuf, maps are encoded as repeated messages:
//
//	message WithMaps {
//	  map<string, int32> string_to_int = 1;
//	  map<int64, string> int_to_string = 2;
//	  map<string, Sample> string_to_msg = 3;
//	}
type WithMaps struct {
	StringToInt map[string]int32   `protobuf:"1,map,string,int32"`
	IntToString map[int64]string   `protobuf:"2,map,int64,string"`
	StringToMsg map[string]*Sample `protobuf:"3,map,string,message"`
}

// BigStruct has many fields including Timeseries
type BigStruct struct {
	Field1  string     `protobuf:"1"`
	Field2  int32      `protobuf:"2"`
	Field3  int64      `protobuf:"3"`
	Field4  bool       `protobuf:"4"`
	Field5  float64    `protobuf:"5"`
	Series  Timeseries `protobuf:"50"` // Timeseries at field 50
	Field51 string     `protobuf:"51"`
	Field52 int32      `protobuf:"52"`
}

// SmallStruct has just 2 fields including the SAME Timeseries type
type SmallStruct struct {
	Name   string     `protobuf:"1"`
	Series Timeseries `protobuf:"2"` // Same Timeseries, different field number
}

// InferredTypes demonstrates simplified tags where types are inferred from Go types.
// No need to specify the protobuf type - it's automatically determined!
type InferredTypes struct {
	// Scalar types - inferred automatically
	Name      string  `protobuf:"1"` // inferred: string
	Age       int32   `protobuf:"2"` // inferred: int32
	Score     float64 `protobuf:"3"` // inferred: double
	IsActive  bool    `protobuf:"4"` // inferred: bool
	Data      []byte  `protobuf:"5"` // inferred: bytes
	BigNum    int64   `protobuf:"6"` // inferred: int64
	SmallNum  float32 `protobuf:"7"` // inferred: float
	Unsigned  uint32  `protobuf:"8"` // inferred: uint32
	BigUnsign uint64  `protobuf:"9"` // inferred: uint64

	// Nested message - inferred automatically
	Inner *Sample `protobuf:"10"` // inferred: message

	// Repeated message - inferred automatically
	Items []Sample `protobuf:"11"` // inferred: message (repeated)

	// Map - key and value types inferred automatically
	Lookup map[string]int32 `protobuf:"12"` // inferred: map<string,int32>

	// Repeated scalars - inferred automatically
	Numbers []int64 `protobuf:"13"` // inferred: int64 (repeated)
}
