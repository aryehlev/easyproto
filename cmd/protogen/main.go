// protogen generates protobuf marshal/unmarshal functions for structs with protobuf tags.
//
// Usage:
//
//	//go:generate go run github.com/VictoriaMetrics/easyproto/cmd/protogen -type=Timeseries,Sample
//
// Struct tags format:
//
//	`protobuf:"fieldNum[,type][,options...]"`
//
// The type is OPTIONAL - it will be inferred from the Go type when omitted:
//
//	string    -> string       int32   -> int32      float32 -> float
//	[]byte    -> bytes        int64   -> int64      float64 -> double
//	bool      -> bool         uint32  -> uint32     CustomType -> message
//	int       -> int64        uint64  -> uint64     map[K]V -> map
//
// Options:
//   - repeated: field is a repeated (slice) field
//   - optional: field is optional (pointer type, nil means unset)
//   - enum: field is an enum type (uses int32 wire type)
//
// When you need non-default wire types, specify explicitly:
//   - sint32, sint64: for signed integers with many negative values
//   - fixed32, fixed64, sfixed32, sfixed64: for fixed-width encoding
//
// Example with inferred types (simple):
//
//	type Timeseries struct {
//	    Name    string   `protobuf:"1"`          // inferred: string
//	    Samples []Sample `protobuf:"2"`          // inferred: message (repeated)
//	}
//
//	type Sample struct {
//	    Value     float64 `protobuf:"1"`         // inferred: double
//	    Timestamp int64   `protobuf:"2"`         // inferred: int64
//	}
//
//	type WithMaps struct {
//	    Data  map[string]int32   `protobuf:"1"` // inferred: map<string,int32>
//	    Items map[string]*Sample `protobuf:"2"` // inferred: map<string,message>
//	}
//
// Example with explicit types (when needed):
//
//	type Explicit struct {
//	    SignedVal int32  `protobuf:"1,sint32"`   // use sint32 encoding
//	    FixedVal  uint64 `protobuf:"2,fixed64"`  // use fixed64 encoding
//	    Status    MyEnum `protobuf:"3,enum"`     // enum type
//	}
package main

import (
	"bytes"
	"flag"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"log"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"text/template"
)

var (
	typeNames = flag.String("type", "", "comma-separated list of type names")
	output    = flag.String("output", "", "output file name; default srcdir/<type>_proto.go")
	noHeader  = flag.Bool("noheader", false, "skip generating the _mp pool and interface definitions (use when adding to existing generated file)")
)

func main() {
	flag.Parse()

	if *typeNames == "" {
		log.Fatal("-type flag is required")
	}

	types := strings.Split(*typeNames, ",")
	for i := range types {
		types[i] = strings.TrimSpace(types[i])
	}

	// Get the directory to parse
	dir := "."
	if len(flag.Args()) > 0 {
		dir = flag.Args()[0]
	}

	// Parse the package
	fset := token.NewFileSet()
	pkgs, err := parser.ParseDir(fset, dir, nil, parser.ParseComments)
	if err != nil {
		log.Fatalf("failed to parse directory %s: %v", dir, err)
	}

	var pkg *ast.Package
	var pkgName string
	for name, p := range pkgs {
		// Skip test packages
		if strings.HasSuffix(name, "_test") {
			continue
		}
		pkg = p
		pkgName = name
		break
	}
	if pkg == nil {
		log.Fatal("no non-test package found")
	}

	// Find the requested types
	typeInfos := make(map[string]*TypeInfo)
	for _, file := range pkg.Files {
		for _, decl := range file.Decls {
			genDecl, ok := decl.(*ast.GenDecl)
			if !ok || genDecl.Tok != token.TYPE {
				continue
			}
			for _, spec := range genDecl.Specs {
				typeSpec, ok := spec.(*ast.TypeSpec)
				if !ok {
					continue
				}
				for _, typeName := range types {
					if typeSpec.Name.Name == typeName {
						structType, ok := typeSpec.Type.(*ast.StructType)
						if !ok {
							log.Fatalf("type %s is not a struct", typeName)
						}
						info, err := parseStruct(typeName, structType)
						if err != nil {
							log.Fatalf("failed to parse struct %s: %v", typeName, err)
						}
						typeInfos[typeName] = info
					}
				}
			}
		}
	}

	// Check all types were found
	for _, typeName := range types {
		if _, ok := typeInfos[typeName]; !ok {
			log.Fatalf("type %s not found", typeName)
		}
	}

	// Generate code
	var buf bytes.Buffer
	if err := generateCode(&buf, pkgName, types, typeInfos, *noHeader); err != nil {
		log.Fatalf("failed to generate code: %v", err)
	}

	// Format the code
	formatted, err := format.Source(buf.Bytes())
	if err != nil {
		// Write unformatted for debugging
		os.WriteFile("debug_unformatted.go", buf.Bytes(), 0644)
		log.Fatalf("failed to format generated code: %v", err)
	}

	// Determine output file
	outputFile := *output
	if outputFile == "" {
		outputFile = filepath.Join(dir, strings.ToLower(types[0])+"_proto.go")
	}

	if err := os.WriteFile(outputFile, formatted, 0644); err != nil {
		log.Fatalf("failed to write output file: %v", err)
	}

	fmt.Printf("Generated %s\n", outputFile)
}

// TypeInfo contains parsed information about a struct type.
type TypeInfo struct {
	Name   string
	Fields []*FieldInfo
}

// FieldInfo contains parsed information about a struct field.
type FieldInfo struct {
	Name          string
	GoType        string
	FieldNum      int
	ProtoType     string
	IsRepeated    bool
	IsMessage     bool
	IsPointer     bool   // Field is a pointer type (*Type)
	IsSliceOfPtr  bool   // Field is a slice of pointers ([]*Type)
	IsOptional    bool   // Field is optional (can be nil/unset)
	IsEnum        bool   // Field is an enum type
	IsMap         bool   // Field is a map type
	IsCustom      bool   // Field uses custom marshaler interface (external types)
	ElemType      string // For slices, the element type (without [] or *)
	RawElemType   string // For slices, the raw element type (with * if applicable)
	BaseType      string // The base type without * or []
	NeedsTypeConv bool   // Needs type conversion (e.g., enum)
	ConvType      string // Type to convert to/from (e.g., int32 for enum)

	// Map-specific fields
	MapKeyType     string // Go type of map key (e.g., "string", "int32")
	MapValueType   string // Go type of map value (e.g., "int32", "*Sample")
	MapKeyProto    string // Proto type of map key (e.g., "string", "int32")
	MapValueProto  string // Proto type of map value (e.g., "int32", "message")
	MapValueIsMsg  bool   // Map value is a message type
	MapValueIsPtr  bool   // Map value is a pointer to message
	MapValueCustom bool   // Map value uses custom marshaler interface
}

func parseStruct(typeName string, structType *ast.StructType) (*TypeInfo, error) {
	info := &TypeInfo{
		Name: typeName,
	}

	// Track field numbers to detect duplicates
	seenFieldNums := make(map[int]string)

	for _, field := range structType.Fields.List {
		if field.Tag == nil {
			continue
		}

		// Parse the struct tag
		tag := reflect.StructTag(strings.Trim(field.Tag.Value, "`"))
		protoTag := tag.Get("protobuf")
		if protoTag == "" {
			continue
		}

		parts := strings.Split(protoTag, ",")
		if len(parts) < 1 {
			return nil, fmt.Errorf("invalid protobuf tag format: %s", protoTag)
		}

		fieldNum, err := strconv.Atoi(strings.TrimSpace(parts[0]))
		if err != nil {
			return nil, fmt.Errorf("invalid field number in tag %q: must be a number", protoTag)
		}

		// Validate field number range (protobuf spec: 1 to 2^29-1, with 19000-19999 reserved)
		if fieldNum < 1 {
			return nil, fmt.Errorf("invalid field number %d in tag %q: must be >= 1", fieldNum, protoTag)
		}
		if fieldNum > 536870911 { // 2^29 - 1
			return nil, fmt.Errorf("invalid field number %d in tag %q: must be <= 536870911", fieldNum, protoTag)
		}
		if fieldNum >= 19000 && fieldNum <= 19999 {
			return nil, fmt.Errorf("invalid field number %d in tag %q: range 19000-19999 is reserved", fieldNum, protoTag)
		}

		// Proto type is optional - can be inferred from Go type
		var protoType string
		if len(parts) >= 2 {
			protoType = strings.TrimSpace(parts[1])
			// Validate explicit protobuf type
			if !isValidProtoType(protoType) {
				return nil, fmt.Errorf("invalid protobuf type %q in tag %q", protoType, protoTag)
			}
		} else {
			// Infer from Go type
			protoType = inferProtoType(field.Type)
		}

		// Reject interface types (like 'any' or custom interfaces)
		if protoType == "interface" {
			fieldName := ""
			if len(field.Names) > 0 {
				fieldName = field.Names[0].Name
			}
			return nil, fmt.Errorf("interface types are not supported for protobuf: field %q in type %s has type %s",
				fieldName, typeName, exprToString(field.Type))
		}

		// Check for options
		isRepeated := false
		isOptional := false
		isEnum := protoType == "enum"
		isMap := protoType == "map"
		isCustom := false

		// For maps, we need key and value types from the tag or infer them
		var mapKeyProto, mapValueProto string
		var mapValueCustom bool
		if isMap {
			if len(parts) >= 4 {
				// Explicit: `protobuf:"1,map,string,int32"`
				mapKeyProto = strings.TrimSpace(parts[2])
				mapValueProto = strings.TrimSpace(parts[3])
			} else if mapType, ok := field.Type.(*ast.MapType); ok {
				// Infer from Go type: `protobuf:"1"` on map[string]int32
				mapKeyProto = inferProtoType(mapType.Key)
				mapValueProto = inferProtoType(mapType.Value)
			}
			// Validate map key type (only certain scalar types allowed)
			if !isValidMapKeyType(mapKeyProto) {
				return nil, fmt.Errorf("invalid map key type %q in tag %q: must be string, bool, or integer type", mapKeyProto, protoTag)
			}
		}

		// Parse options from remaining parts (if any)
		optionStart := 2
		if isMap && len(parts) >= 4 {
			optionStart = 4 // Skip map key/value types
		}
		if len(parts) > optionStart {
			for _, part := range parts[optionStart:] {
				switch strings.TrimSpace(part) {
				case "repeated":
					isRepeated = true
				case "optional":
					isOptional = true
				case "enum":
					isEnum = true
				case "custom":
					isCustom = true
					// For maps, custom applies to the value type
					if isMap {
						mapValueCustom = true
					}
				}
			}
		}

		// Handle embedded fields (anonymous fields) - they have no Names
		fieldNames := make([]string, 0, len(field.Names))
		for _, name := range field.Names {
			fieldNames = append(fieldNames, name.Name)
		}
		if len(fieldNames) == 0 {
			// Embedded field - use the type name as the field name
			embeddedName := getTypeName(field.Type)
			if embeddedName == "" {
				return nil, fmt.Errorf("cannot determine name for embedded field with tag %q in type %s", protoTag, typeName)
			}
			fieldNames = append(fieldNames, embeddedName)
		}

		for _, fieldName := range fieldNames {
			// Check for duplicate field numbers
			if existingField, ok := seenFieldNums[fieldNum]; ok {
				return nil, fmt.Errorf("duplicate field number %d: used by both %q and %q in type %s",
					fieldNum, existingField, fieldName, typeName)
			}
			seenFieldNums[fieldNum] = fieldName

			fi := &FieldInfo{
				Name:       fieldName,
				FieldNum:   fieldNum,
				ProtoType:  protoType,
				IsRepeated: isRepeated,
				IsOptional: isOptional,
				IsMessage:  protoType == "message",
				IsEnum:     isEnum,
				IsMap:      isMap,
				IsCustom:   isCustom,
			}

			// Analyze Go type
			fi.GoType = exprToString(field.Type)
			analyzeType(fi, field.Type)

			// Handle map-specific parsing
			if fi.IsMap {
				fi.MapKeyProto = mapKeyProto
				fi.MapValueProto = mapValueProto
				fi.MapValueIsMsg = mapValueProto == "message"
				fi.MapValueCustom = mapValueCustom
				// Extract key/value Go types from the AST
				if mapType, ok := field.Type.(*ast.MapType); ok {
					fi.MapKeyType = exprToString(mapType.Key)
					fi.MapValueType = exprToString(mapType.Value)
					// Check if value is a pointer
					if _, isPtr := mapType.Value.(*ast.StarExpr); isPtr {
						fi.MapValueIsPtr = true
					}
				}
			}

			// Handle enum type conversion
			if fi.IsEnum {
				fi.NeedsTypeConv = true
				fi.ConvType = "int32"
			}

			info.Fields = append(info.Fields, fi)
		}
	}

	// Sort fields by field number
	sort.Slice(info.Fields, func(i, j int) bool {
		return info.Fields[i].FieldNum < info.Fields[j].FieldNum
	})

	return info, nil
}

// validProtoTypes is the set of valid protobuf types
var validProtoTypes = map[string]bool{
	"string":   true,
	"bytes":    true,
	"int32":    true,
	"int64":    true,
	"uint32":   true,
	"uint64":   true,
	"sint32":   true,
	"sint64":   true,
	"bool":     true,
	"double":   true,
	"float":    true,
	"fixed32":  true,
	"fixed64":  true,
	"sfixed32": true,
	"sfixed64": true,
	"message":  true,
	"enum":     true,
	"map":      true,
}

// validMapKeyTypes is the set of valid protobuf map key types
var validMapKeyTypes = map[string]bool{
	"string": true,
	"int32":  true,
	"int64":  true,
	"uint32": true,
	"uint64": true,
	"sint32": true,
	"sint64": true,
	"bool":   true,
	// Note: float/double and bytes are NOT valid map key types in protobuf
}

// isValidProtoType checks if a protobuf type is valid
func isValidProtoType(protoType string) bool {
	return validProtoTypes[protoType]
}

// isValidMapKeyType checks if a protobuf type is valid as a map key
func isValidMapKeyType(protoType string) bool {
	return validMapKeyTypes[protoType]
}

// getTypeName extracts the type name from an AST expression (for embedded fields)
func getTypeName(expr ast.Expr) string {
	switch t := expr.(type) {
	case *ast.Ident:
		return t.Name
	case *ast.SelectorExpr:
		return t.Sel.Name
	case *ast.StarExpr:
		return getTypeName(t.X)
	default:
		return ""
	}
}

// inferProtoType infers the protobuf type from a Go AST type expression.
// This allows users to omit the type in the tag: `protobuf:"1"` instead of `protobuf:"1,string"`
func inferProtoType(expr ast.Expr) string {
	switch t := expr.(type) {
	case *ast.Ident:
		// Basic Go types
		switch t.Name {
		case "string":
			return "string"
		case "bool":
			return "bool"
		case "int32":
			return "int32"
		case "int64":
			return "int64"
		case "int": // Go int -> int64 (safest default)
			return "int64"
		case "uint32":
			return "uint32"
		case "uint64":
			return "uint64"
		case "uint": // Go uint -> uint64 (safest default)
			return "uint64"
		case "float32":
			return "float"
		case "float64":
			return "double"
		case "byte": // single byte is unusual, treat as int32
			return "int32"
		case "any": // interface{} type - not supported
			return "interface"
		default:
			// Custom type - assume it's a message
			return "message"
		}
	case *ast.InterfaceType:
		// Interface types are not supported
		return "interface"
	case *ast.SelectorExpr:
		// Qualified type like time.Time - treat as message
		return "message"
	case *ast.StarExpr:
		// Pointer type - infer from underlying type
		return inferProtoType(t.X)
	case *ast.ArrayType:
		if t.Len == nil { // slice
			// Check for []byte
			if ident, ok := t.Elt.(*ast.Ident); ok && ident.Name == "byte" {
				return "bytes"
			}
			// For other slices, infer element type
			return inferProtoType(t.Elt)
		}
		// Fixed-size arrays - treat element type
		return inferProtoType(t.Elt)
	case *ast.MapType:
		return "map"
	default:
		return "bytes" // fallback
	}
}

func analyzeType(fi *FieldInfo, expr ast.Expr) {
	switch t := expr.(type) {
	case *ast.Ident:
		fi.BaseType = t.Name
		fi.ElemType = t.Name
		fi.RawElemType = t.Name
	case *ast.SelectorExpr:
		fullType := exprToString(t)
		fi.BaseType = fullType
		fi.ElemType = fullType
		fi.RawElemType = fullType
	case *ast.StarExpr:
		fi.IsPointer = true
		fi.IsOptional = true
		inner := exprToString(t.X)
		fi.BaseType = inner
		fi.ElemType = inner
		fi.RawElemType = inner
	case *ast.ArrayType:
		if t.Len == nil { // slice
			// Special case: []byte is NOT a repeated field, it's a scalar bytes type
			if ident, ok := t.Elt.(*ast.Ident); ok && ident.Name == "byte" {
				fi.BaseType = "[]byte"
				fi.ElemType = "byte"
				fi.RawElemType = "byte"
				// Don't set IsRepeated for []byte - it's handled as scalar bytes
				return
			}

			fi.IsRepeated = true
			// Check if it's a slice of pointers
			if star, ok := t.Elt.(*ast.StarExpr); ok {
				fi.IsSliceOfPtr = true
				fi.ElemType = exprToString(star.X)
				fi.RawElemType = "*" + fi.ElemType
				fi.BaseType = fi.ElemType
			} else {
				fi.ElemType = exprToString(t.Elt)
				fi.RawElemType = fi.ElemType
				fi.BaseType = fi.ElemType
			}
		}
	}
}

func exprToString(expr ast.Expr) string {
	switch t := expr.(type) {
	case *ast.Ident:
		return t.Name
	case *ast.SelectorExpr:
		return exprToString(t.X) + "." + t.Sel.Name
	case *ast.StarExpr:
		return "*" + exprToString(t.X)
	case *ast.ArrayType:
		if t.Len == nil {
			return "[]" + exprToString(t.Elt)
		}
		return fmt.Sprintf("[%s]%s", exprToString(t.Len), exprToString(t.Elt))
	case *ast.BasicLit:
		return t.Value
	case *ast.MapType:
		return fmt.Sprintf("map[%s]%s", exprToString(t.Key), exprToString(t.Value))
	default:
		return fmt.Sprintf("%T", expr)
	}
}

const codeTemplate = `// Code generated by protogen. DO NOT EDIT.

package {{.PackageName}}

import (
	"fmt"

	"github.com/VictoriaMetrics/easyproto"
)
{{if not .NoHeader}}
var _mp easyproto.MarshalerPool

// ProtobufMarshaler is the interface for types that can marshal to protobuf.
// Implement this interface to use custom types as nested messages.
type ProtobufMarshaler interface {
	MarshalProtobufTo(mm *easyproto.MessageMarshaler)
}

// ProtobufUnmarshaler is the interface for types that can unmarshal from protobuf.
// Implement this interface to use custom types as nested messages.
type ProtobufUnmarshaler interface {
	UnmarshalProtobuf(src []byte) error
}
{{end}}
{{range $typeName := .TypeNames}}
{{$info := index $.TypeInfos $typeName}}
// MarshalProtobuf marshals {{$info.Name}} into protobuf message, appends this message to dst and returns the result.
//
// This function doesn't allocate memory on repeated calls.
func (x *{{$info.Name}}) MarshalProtobuf(dst []byte) []byte {
	m := _mp.Get()
	x.MarshalProtobufTo(m.MessageMarshaler())
	dst = m.Marshal(dst)
	_mp.Put(m)
	return dst
}

// MarshalProtobufTo marshals {{$info.Name}} fields to the given MessageMarshaler.
// Implements ProtobufMarshaler interface.
func (x *{{$info.Name}}) MarshalProtobufTo(mm *easyproto.MessageMarshaler) {
{{- range $field := $info.Fields}}
{{marshalField $info $field}}
{{- end}}
}

// UnmarshalProtobuf unmarshals {{$info.Name}} from protobuf message at src.
func (x *{{$info.Name}}) UnmarshalProtobuf(src []byte) (err error) {
	// Set default values
{{- range $field := $info.Fields}}
{{resetField $info $field}}
{{- end}}

	// Parse message
	var fc easyproto.FieldContext
	for len(src) > 0 {
		src, err = fc.NextField(src)
		if err != nil {
			return fmt.Errorf("cannot read next field in {{$info.Name}}: %w", err)
		}
		switch fc.FieldNum {
{{- range $field := $info.Fields}}
		case {{$field.FieldNum}}:
{{unmarshalField $info $field}}
{{- end}}
		}
	}
	return nil
}

{{end}}
`

func generateCode(buf *bytes.Buffer, pkgName string, typeNames []string, typeInfos map[string]*TypeInfo, skipHeader bool) error {
	funcMap := template.FuncMap{
		"appendFunc":     appendFunc,
		"readFunc":       readFunc,
		"unpackFunc":     unpackFunc,
		"zeroValue":      zeroValue,
		"marshalField":   marshalField,
		"unmarshalField": unmarshalField,
		"resetField":     resetField,
	}

	tmpl, err := template.New("code").Funcs(funcMap).Parse(codeTemplate)
	if err != nil {
		return fmt.Errorf("failed to parse template: %w", err)
	}

	data := struct {
		PackageName string
		TypeNames   []string
		TypeInfos   map[string]*TypeInfo
		NoHeader    bool
	}{
		PackageName: pkgName,
		TypeNames:   typeNames,
		TypeInfos:   typeInfos,
		NoHeader:    skipHeader,
	}

	return tmpl.Execute(buf, data)
}

// marshalField generates marshal code for a single field.
func marshalField(info *TypeInfo, field *FieldInfo) string {
	var buf strings.Builder
	fieldAccess := "x." + field.Name

	if field.IsMap {
		// Maps are encoded as repeated messages with key (field 1) and value (field 2)
		buf.WriteString(fmt.Sprintf("\tfor k, v := range %s {\n", fieldAccess))
		buf.WriteString(fmt.Sprintf("\t\tmm2 := mm.AppendMessage(%d)\n", field.FieldNum))

		// Marshal key (field 1)
		buf.WriteString(fmt.Sprintf("\t\tmm2.%s(1, k)\n", appendFunc(field.MapKeyProto, false)))

		// Marshal value (field 2)
		if field.MapValueIsMsg {
			if field.MapValueIsPtr {
				buf.WriteString("\t\tif v != nil {\n")
				buf.WriteString("\t\t\tv.MarshalProtobufTo(mm2.AppendMessage(2))\n")
				buf.WriteString("\t\t}\n")
			} else {
				buf.WriteString("\t\tv.MarshalProtobufTo(mm2.AppendMessage(2))\n")
			}
		} else {
			buf.WriteString(fmt.Sprintf("\t\tmm2.%s(2, v)\n", appendFunc(field.MapValueProto, false)))
		}

		buf.WriteString("\t}")
		return buf.String()
	}

	if field.IsMessage {
		if field.IsPointer && !field.IsRepeated {
			// Optional message pointer: *Message
			buf.WriteString(fmt.Sprintf("\tif %s != nil {\n", fieldAccess))
			buf.WriteString(fmt.Sprintf("\t\t%s.MarshalProtobufTo(mm.AppendMessage(%d))\n", fieldAccess, field.FieldNum))
			buf.WriteString("\t}")
		} else if field.IsRepeated && field.IsSliceOfPtr {
			// Slice of message pointers: []*Message
			buf.WriteString(fmt.Sprintf("\tfor _, item := range %s {\n", fieldAccess))
			buf.WriteString("\t\tif item != nil {\n")
			buf.WriteString(fmt.Sprintf("\t\t\titem.MarshalProtobufTo(mm.AppendMessage(%d))\n", field.FieldNum))
			buf.WriteString("\t\t}\n")
			buf.WriteString("\t}")
		} else if field.IsRepeated {
			// Slice of messages: []Message
			buf.WriteString(fmt.Sprintf("\tfor i := range %s {\n", fieldAccess))
			buf.WriteString(fmt.Sprintf("\t\t%s[i].MarshalProtobufTo(mm.AppendMessage(%d))\n", fieldAccess, field.FieldNum))
			buf.WriteString("\t}")
		} else {
			// Embedded message: Message
			buf.WriteString(fmt.Sprintf("\t%s.MarshalProtobufTo(mm.AppendMessage(%d))", fieldAccess, field.FieldNum))
		}
	} else if field.IsEnum {
		if field.IsPointer && !field.IsRepeated {
			// Optional enum pointer
			buf.WriteString(fmt.Sprintf("\tif %s != nil {\n", fieldAccess))
			buf.WriteString(fmt.Sprintf("\t\tmm.AppendInt32(%d, int32(*%s))\n", field.FieldNum, fieldAccess))
			buf.WriteString("\t}")
		} else if field.IsRepeated {
			// Repeated enum
			buf.WriteString(fmt.Sprintf("\tfor _, v := range %s {\n", fieldAccess))
			buf.WriteString(fmt.Sprintf("\t\tmm.AppendInt32(%d, int32(v))\n", field.FieldNum))
			buf.WriteString("\t}")
		} else {
			// Regular enum
			buf.WriteString(fmt.Sprintf("\tmm.AppendInt32(%d, int32(%s))", field.FieldNum, fieldAccess))
		}
	} else if field.IsPointer && !field.IsRepeated {
		// Optional scalar pointer
		buf.WriteString(fmt.Sprintf("\tif %s != nil {\n", fieldAccess))
		buf.WriteString(fmt.Sprintf("\t\tmm.%s(%d, *%s)\n", appendFunc(field.ProtoType, false), field.FieldNum, fieldAccess))
		buf.WriteString("\t}")
	} else if field.IsRepeated {
		// Repeated scalar
		appendFn := appendFunc(field.ProtoType, true)
		buf.WriteString(fmt.Sprintf("\tmm.%s(%d, %s)", appendFn, field.FieldNum, fieldAccess))
	} else {
		// Regular scalar
		buf.WriteString(fmt.Sprintf("\tmm.%s(%d, %s)", appendFunc(field.ProtoType, false), field.FieldNum, fieldAccess))
	}

	return buf.String()
}

// unmarshalField generates unmarshal code for a single field.
func unmarshalField(info *TypeInfo, field *FieldInfo) string {
	var buf strings.Builder
	fieldAccess := "x." + field.Name

	if field.IsMap {
		// Maps are encoded as repeated messages with key (field 1) and value (field 2)
		buf.WriteString("\t\t\tdata, ok := fc.MessageData()\n")
		buf.WriteString("\t\t\tif !ok {\n")
		buf.WriteString(fmt.Sprintf("\t\t\t\treturn fmt.Errorf(\"cannot read %s.%s data\")\n", info.Name, field.Name))
		buf.WriteString("\t\t\t}\n")

		// Parse the map entry message
		buf.WriteString(fmt.Sprintf("\t\t\tvar mk %s\n", field.MapKeyType))
		buf.WriteString(fmt.Sprintf("\t\t\tvar mv %s\n", field.MapValueType))
		buf.WriteString("\t\t\tvar fc2 easyproto.FieldContext\n")
		buf.WriteString("\t\t\tfor len(data) > 0 {\n")
		buf.WriteString("\t\t\t\tdata, err = fc2.NextField(data)\n")
		buf.WriteString("\t\t\t\tif err != nil {\n")
		buf.WriteString(fmt.Sprintf("\t\t\t\t\treturn fmt.Errorf(\"cannot read %s.%s entry: %%w\", err)\n", info.Name, field.Name))
		buf.WriteString("\t\t\t\t}\n")
		buf.WriteString("\t\t\t\tswitch fc2.FieldNum {\n")

		// Key (field 1)
		buf.WriteString("\t\t\t\tcase 1:\n")
		buf.WriteString(fmt.Sprintf("\t\t\t\t\tkv, ok := fc2.%s()\n", readFunc(field.MapKeyProto)))
		buf.WriteString("\t\t\t\t\tif !ok {\n")
		buf.WriteString(fmt.Sprintf("\t\t\t\t\t\treturn fmt.Errorf(\"cannot read %s.%s key\")\n", info.Name, field.Name))
		buf.WriteString("\t\t\t\t\t}\n")
		buf.WriteString("\t\t\t\t\tmk = kv\n")

		// Value (field 2)
		buf.WriteString("\t\t\t\tcase 2:\n")
		if field.MapValueIsMsg {
			buf.WriteString("\t\t\t\t\tvdata, ok := fc2.MessageData()\n")
			buf.WriteString("\t\t\t\t\tif !ok {\n")
			buf.WriteString(fmt.Sprintf("\t\t\t\t\t\treturn fmt.Errorf(\"cannot read %s.%s value data\")\n", info.Name, field.Name))
			buf.WriteString("\t\t\t\t\t}\n")
			if field.MapValueIsPtr {
				// Extract base type from pointer type (e.g., "*Sample" -> "Sample")
				baseValueType := strings.TrimPrefix(field.MapValueType, "*")
				buf.WriteString(fmt.Sprintf("\t\t\t\t\tmv = &%s{}\n", baseValueType))
			}
			buf.WriteString("\t\t\t\t\tif err := mv.UnmarshalProtobuf(vdata); err != nil {\n")
			buf.WriteString(fmt.Sprintf("\t\t\t\t\t\treturn fmt.Errorf(\"cannot unmarshal %s.%s value: %%w\", err)\n", info.Name, field.Name))
			buf.WriteString("\t\t\t\t\t}\n")
		} else {
			buf.WriteString(fmt.Sprintf("\t\t\t\t\tvv, ok := fc2.%s()\n", readFunc(field.MapValueProto)))
			buf.WriteString("\t\t\t\t\tif !ok {\n")
			buf.WriteString(fmt.Sprintf("\t\t\t\t\t\treturn fmt.Errorf(\"cannot read %s.%s value\")\n", info.Name, field.Name))
			buf.WriteString("\t\t\t\t\t}\n")
			buf.WriteString("\t\t\t\t\tmv = vv\n")
		}

		buf.WriteString("\t\t\t\t}\n") // end switch
		buf.WriteString("\t\t\t}\n")   // end for

		// Initialize map if nil and add entry
		buf.WriteString(fmt.Sprintf("\t\t\tif %s == nil {\n", fieldAccess))
		buf.WriteString(fmt.Sprintf("\t\t\t\t%s = make(%s)\n", fieldAccess, field.GoType))
		buf.WriteString("\t\t\t}\n")
		buf.WriteString(fmt.Sprintf("\t\t\t%s[mk] = mv", fieldAccess))

		return buf.String()
	}

	if field.IsMessage {
		buf.WriteString("\t\t\tdata, ok := fc.MessageData()\n")
		buf.WriteString("\t\t\tif !ok {\n")
		buf.WriteString(fmt.Sprintf("\t\t\t\treturn fmt.Errorf(\"cannot read %s.%s data\")\n", info.Name, field.Name))
		buf.WriteString("\t\t\t}\n")

		if field.IsPointer && !field.IsRepeated {
			// Optional message pointer: *Message
			buf.WriteString(fmt.Sprintf("\t\t\tif %s == nil {\n", fieldAccess))
			buf.WriteString(fmt.Sprintf("\t\t\t\t%s = &%s{}\n", fieldAccess, field.ElemType))
			buf.WriteString("\t\t\t}\n")
			buf.WriteString(fmt.Sprintf("\t\t\tif err := %s.UnmarshalProtobuf(data); err != nil {\n", fieldAccess))
			buf.WriteString(fmt.Sprintf("\t\t\t\treturn fmt.Errorf(\"cannot unmarshal %s.%s: %%w\", err)\n", info.Name, field.Name))
			buf.WriteString("\t\t\t}")
		} else if field.IsRepeated && field.IsSliceOfPtr {
			// Slice of message pointers: []*Message
			buf.WriteString(fmt.Sprintf("\t\t\titem := &%s{}\n", field.ElemType))
			buf.WriteString("\t\t\tif err := item.UnmarshalProtobuf(data); err != nil {\n")
			buf.WriteString(fmt.Sprintf("\t\t\t\treturn fmt.Errorf(\"cannot unmarshal %s.%s: %%w\", err)\n", info.Name, field.Name))
			buf.WriteString("\t\t\t}\n")
			buf.WriteString(fmt.Sprintf("\t\t\t%s = append(%s, item)", fieldAccess, fieldAccess))
		} else if field.IsRepeated {
			// Slice of messages: []Message
			buf.WriteString(fmt.Sprintf("\t\t\t%s = append(%s, %s{})\n", fieldAccess, fieldAccess, field.ElemType))
			buf.WriteString(fmt.Sprintf("\t\t\titem := &%s[len(%s)-1]\n", fieldAccess, fieldAccess))
			buf.WriteString("\t\t\tif err := item.UnmarshalProtobuf(data); err != nil {\n")
			buf.WriteString(fmt.Sprintf("\t\t\t\treturn fmt.Errorf(\"cannot unmarshal %s.%s: %%w\", err)\n", info.Name, field.Name))
			buf.WriteString("\t\t\t}")
		} else {
			// Embedded message: Message
			buf.WriteString(fmt.Sprintf("\t\t\tif err := %s.UnmarshalProtobuf(data); err != nil {\n", fieldAccess))
			buf.WriteString(fmt.Sprintf("\t\t\t\treturn fmt.Errorf(\"cannot unmarshal %s.%s: %%w\", err)\n", info.Name, field.Name))
			buf.WriteString("\t\t\t}")
		}
	} else if field.IsEnum {
		if field.IsPointer && !field.IsRepeated {
			// Optional enum pointer
			buf.WriteString("\t\t\tv, ok := fc.Int32()\n")
			buf.WriteString("\t\t\tif !ok {\n")
			buf.WriteString(fmt.Sprintf("\t\t\t\treturn fmt.Errorf(\"cannot read %s.%s\")\n", info.Name, field.Name))
			buf.WriteString("\t\t\t}\n")
			buf.WriteString(fmt.Sprintf("\t\t\ttmp := %s(v)\n", field.ElemType))
			buf.WriteString(fmt.Sprintf("\t\t\t%s = &tmp", fieldAccess))
		} else if field.IsRepeated {
			// Repeated enum - handle both packed and unpacked
			buf.WriteString("\t\t\tif v, ok := fc.Int32(); ok {\n")
			buf.WriteString(fmt.Sprintf("\t\t\t\t%s = append(%s, %s(v))\n", fieldAccess, fieldAccess, field.ElemType))
			buf.WriteString("\t\t\t} else {\n")
			buf.WriteString(fmt.Sprintf("\t\t\t\treturn fmt.Errorf(\"cannot read %s.%s\")\n", info.Name, field.Name))
			buf.WriteString("\t\t\t}")
		} else {
			// Regular enum
			buf.WriteString("\t\t\tv, ok := fc.Int32()\n")
			buf.WriteString("\t\t\tif !ok {\n")
			buf.WriteString(fmt.Sprintf("\t\t\t\treturn fmt.Errorf(\"cannot read %s.%s\")\n", info.Name, field.Name))
			buf.WriteString("\t\t\t}\n")
			buf.WriteString(fmt.Sprintf("\t\t\t%s = %s(v)", fieldAccess, field.BaseType))
		}
	} else if field.IsPointer && !field.IsRepeated {
		// Optional scalar pointer
		buf.WriteString(fmt.Sprintf("\t\t\tv, ok := fc.%s()\n", readFunc(field.ProtoType)))
		buf.WriteString("\t\t\tif !ok {\n")
		buf.WriteString(fmt.Sprintf("\t\t\t\treturn fmt.Errorf(\"cannot read %s.%s\")\n", info.Name, field.Name))
		buf.WriteString("\t\t\t}\n")
		buf.WriteString(fmt.Sprintf("\t\t\t%s = &v", fieldAccess))
	} else if field.IsRepeated {
		// Repeated scalar
		buf.WriteString("\t\t\tvar ok bool\n")
		buf.WriteString(fmt.Sprintf("\t\t\t%s, ok = fc.%s(%s)\n", fieldAccess, unpackFunc(field.ProtoType), fieldAccess))
		buf.WriteString("\t\t\tif !ok {\n")
		buf.WriteString(fmt.Sprintf("\t\t\t\treturn fmt.Errorf(\"cannot read %s.%s\")\n", info.Name, field.Name))
		buf.WriteString("\t\t\t}")
	} else {
		// Regular scalar
		buf.WriteString(fmt.Sprintf("\t\t\tv, ok := fc.%s()\n", readFunc(field.ProtoType)))
		buf.WriteString("\t\t\tif !ok {\n")
		buf.WriteString(fmt.Sprintf("\t\t\t\treturn fmt.Errorf(\"cannot read %s.%s\")\n", info.Name, field.Name))
		buf.WriteString("\t\t\t}\n")
		buf.WriteString(fmt.Sprintf("\t\t\t%s = v", fieldAccess))
	}

	return buf.String()
}

// resetField generates reset code for a single field.
func resetField(info *TypeInfo, field *FieldInfo) string {
	fieldAccess := "x." + field.Name

	if field.IsMap {
		// Clear map by deleting all keys (preserves allocated memory)
		return fmt.Sprintf("\tfor k := range %s {\n\t\tdelete(%s, k)\n\t}", fieldAccess, fieldAccess)
	}

	if field.IsRepeated {
		return fmt.Sprintf("\t%s = %s[:0]", fieldAccess, fieldAccess)
	}

	if field.IsPointer {
		return fmt.Sprintf("\t%s = nil", fieldAccess)
	}

	// Enums are numeric types and should be reset to 0
	if field.IsEnum {
		return fmt.Sprintf("\t%s = 0", fieldAccess)
	}

	return fmt.Sprintf("\t%s = %s", fieldAccess, zeroValue(field.GoType))
}

// appendFunc returns the MessageMarshaler append function name for a protobuf type.
func appendFunc(protoType string, isRepeated bool) string {
	if isRepeated {
		switch protoType {
		case "int32":
			return "AppendInt32s"
		case "int64":
			return "AppendInt64s"
		case "uint32":
			return "AppendUint32s"
		case "uint64":
			return "AppendUint64s"
		case "sint32":
			return "AppendSint32s"
		case "sint64":
			return "AppendSint64s"
		case "bool":
			return "AppendBools"
		case "double":
			return "AppendDoubles"
		case "float":
			return "AppendFloats"
		case "fixed32":
			return "AppendFixed32s"
		case "fixed64":
			return "AppendFixed64s"
		case "sfixed32":
			return "AppendSfixed32s"
		case "sfixed64":
			return "AppendSfixed64s"
		}
	}

	switch protoType {
	case "string":
		return "AppendString"
	case "bytes":
		return "AppendBytes"
	case "int32", "enum":
		return "AppendInt32"
	case "int64":
		return "AppendInt64"
	case "uint32":
		return "AppendUint32"
	case "uint64":
		return "AppendUint64"
	case "sint32":
		return "AppendSint32"
	case "sint64":
		return "AppendSint64"
	case "bool":
		return "AppendBool"
	case "double":
		return "AppendDouble"
	case "float":
		return "AppendFloat"
	case "fixed32":
		return "AppendFixed32"
	case "fixed64":
		return "AppendFixed64"
	case "sfixed32":
		return "AppendSfixed32"
	case "sfixed64":
		return "AppendSfixed64"
	default:
		return "AppendBytes" // fallback
	}
}

// readFunc returns the FieldContext read function name for a protobuf type.
func readFunc(protoType string) string {
	switch protoType {
	case "string":
		return "String"
	case "bytes":
		return "Bytes"
	case "int32", "enum":
		return "Int32"
	case "int64":
		return "Int64"
	case "uint32":
		return "Uint32"
	case "uint64":
		return "Uint64"
	case "sint32":
		return "Sint32"
	case "sint64":
		return "Sint64"
	case "bool":
		return "Bool"
	case "double":
		return "Double"
	case "float":
		return "Float"
	case "fixed32":
		return "Fixed32"
	case "fixed64":
		return "Fixed64"
	case "sfixed32":
		return "Sfixed32"
	case "sfixed64":
		return "Sfixed64"
	default:
		return "Bytes" // fallback
	}
}

// unpackFunc returns the FieldContext unpack function name for packed repeated fields.
func unpackFunc(protoType string) string {
	switch protoType {
	case "int32", "enum":
		return "UnpackInt32s"
	case "int64":
		return "UnpackInt64s"
	case "uint32":
		return "UnpackUint32s"
	case "uint64":
		return "UnpackUint64s"
	case "sint32":
		return "UnpackSint32s"
	case "sint64":
		return "UnpackSint64s"
	case "bool":
		return "UnpackBools"
	case "double":
		return "UnpackDoubles"
	case "float":
		return "UnpackFloats"
	case "fixed32":
		return "UnpackFixed32s"
	case "fixed64":
		return "UnpackFixed64s"
	case "sfixed32":
		return "UnpackSfixed32s"
	case "sfixed64":
		return "UnpackSfixed64s"
	default:
		return "Bytes" // fallback
	}
}

// zeroValue returns the zero value for a Go type.
func zeroValue(goType string) string {
	switch goType {
	case "string":
		return `""`
	case "bool":
		return "false"
	case "int", "int8", "int16", "int32", "int64":
		return "0"
	case "uint", "uint8", "uint16", "uint32", "uint64":
		return "0"
	case "float32", "float64":
		return "0"
	case "[]byte":
		return "nil"
	default:
		// For pointer types
		if strings.HasPrefix(goType, "*") {
			return "nil"
		}
		// For slice types
		if strings.HasPrefix(goType, "[]") {
			return "nil"
		}
		// For map types
		if strings.HasPrefix(goType, "map[") {
			return "nil"
		}
		// For custom types (enums, etc.), use zero literal
		return goType + "{}"
	}
}
